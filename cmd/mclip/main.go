package main

import (
	"bytes"
	"context"
	"crypto/md5"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"text/template"
	"time"

	"github.com/thismixer/MultiClip/internal/clipboard"
	"github.com/thismixer/MultiClip/internal/network"
)

var linuxTemplate string

var macTemplate string

func main() {
	verbose := flag.Bool("v", false, "запустить с выводом логов в терминал")
	stop := flag.Bool("stop", false, "остановить фоновый процесс mclip")
	install := flag.Bool("install", false, "добавить программу в автозагрузку системы")
	flag.Parse()

	if *install {
		installService()
		return
	}

	if *stop {
		stopDaemon()
		return
	}

	if !*verbose && os.Getenv("MCLIP_DAEMON") != "1" {
		startDaemon()
		return
	}

	runApp(*verbose)
}

func runApp(isVerbose bool) {
	cb := clipboard.New()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var (
		lastText      string
		lastImageHash [16]byte
		remotes       sync.Map
		mu            sync.Mutex
	)

	addPeer := func(addr string) {
		if _, loaded := remotes.LoadOrStore(addr, true); !loaded {
			if isVerbose {
				fmt.Printf("[+] Соединение установлено: %s\n", addr)
			}
		}
	}

	go network.StartServer(cb, "8080", func(text string) {
		mu.Lock()
		lastText = text
		mu.Unlock()

		if isVerbose {
			fmt.Printf("<- Получен текст: %s\n", limitString(text, 50))
		}
	}, func(imgData []byte) {
		mu.Lock()
		lastImageHash = md5.Sum(imgData)
		mu.Unlock()
		if isVerbose {
			fmt.Println("<- Получено изображение")
		}
	}, addPeer)

	go network.Advertise(ctx, 8080)
	go network.Discover(ctx, addPeer)

	if isVerbose {
		fmt.Println("MultiClip запущен. Нажмите Ctrl+C для выхода.")
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			time.Sleep(1000 * time.Millisecond)

			mu.Lock()
			currentImg, errImg := cb.GetImage()
			if errImg == nil && len(currentImg) > 0 {
				currentHash := md5.Sum(currentImg)
				if currentHash != lastImageHash {
					lastImageHash = currentHash
					if isVerbose {
						fmt.Println("-> Отправка изображения...")
					}
					broadcast(&remotes, "", currentImg, isVerbose)
					mu.Unlock()
					continue
				}
			}

			currentText, errText := cb.GetText()
			if errText == nil && currentText != lastText && currentText != "" {
				lastText = currentText
				if isVerbose {
					fmt.Printf("-> Отправка текста: %s\n", limitString(currentText, 50))
				}
				broadcast(&remotes, currentText, nil, isVerbose)
			}
			mu.Unlock()
		}
	}
}

func installService() {
	executable, _ := os.Executable()
	home, _ := os.UserHomeDir()

	data := struct{ Executable string }{
		Executable: executable,
	}

	var targetPath string
	var tplContent string

	switch runtime.GOOS {
	case "darwin":
		targetPath = filepath.Join(home, "Library/LaunchAgents/com.thismixer.mclip.plist")
		tplContent = macTemplate
	case "linux":
		targetPath = filepath.Join(home, ".config/systemd/user/mclip.service")
		tplContent = linuxTemplate
	default:
		fmt.Println("[-] ОС не поддерживается для автоустановки.")
		return
	}

	tmpl, err := template.New("install").Parse(tplContent)
	if err != nil {
		fmt.Printf("[-] Ошибка шаблона: %v\n", err)
		return
	}

	var buf bytes.Buffer
	tmpl.Execute(&buf, data)

	os.MkdirAll(filepath.Dir(targetPath), 0755)
	err = os.WriteFile(targetPath, buf.Bytes(), 0644)
	if err != nil {
		fmt.Printf("[-] Ошибка записи: %v\n", err)
		return
	}

	if runtime.GOOS == "darwin" {
		exec.Command("launchctl", "unload", targetPath).Run()
		exec.Command("launchctl", "load", targetPath).Run()
	} else {
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		exec.Command("systemctl", "--user", "enable", "mclip", "--now").Run()
	}

	fmt.Printf("[+] Успешно! Программа добавлена в автозагрузку: %s\n", targetPath)
}

func startDaemon() {
	cmd := exec.Command(os.Args[0], "-v")
	cmd.Env = append(os.Environ(), "MCLIP_DAEMON=1")

	if err := cmd.Start(); err != nil {
		fmt.Printf("[-] Ошибка запуска демона: %v\n", err)
		return
	}
	fmt.Printf("[+] MultiClip работает в фоне (PID: %d)\n", cmd.Process.Pid)
	os.Exit(0)
}

func stopDaemon() {
	cmd := exec.Command("pkill", "-f", os.Args[0])
	if err := cmd.Run(); err != nil {
		fmt.Println("[-] Процесс не найден.")
	} else {
		fmt.Println("[!] MultiClip остановлен.")
	}
}

func limitString(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return string(runes)
}

func broadcast(remotes *sync.Map, text string, img []byte, isVerbose bool) {
	remotes.Range(func(key, value any) bool {
		addr := key.(string)
		go func(address string) {
			var err error
			if text != "" {
				err = network.SendText(address, text)
			} else if len(img) > 0 {
				err = network.SendImage(address, img)
			}
			if err != nil {
				remotes.Delete(address)
				if isVerbose {
					fmt.Printf("[-] Связь с %s потеряна\n", address)
				}
			}
		}(addr)
		return true
	})
}
