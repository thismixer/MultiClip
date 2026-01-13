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

var activePeers sync.Map

func main() {
	verbose := flag.Bool("v", false, "запустить с выводом логов в терминал")
	stop := flag.Bool("stop", false, "остановить фоновый процесс mclip")
	install := flag.Bool("install", false, "добавить программу в автозагрузку")
	uninstall := flag.Bool("uninstall", false, "убрать программу из автозагрузки")
	devices := flag.Bool("devices", false, "показать список подключенных устройств")
	flag.Parse()

	if *uninstall {
		uninstallService()
		return
	}

	if *devices {
		showDevices()
		return
	}

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

func uninstallService() {
	home, _ := os.UserHomeDir()
	var targetPath string

	switch runtime.GOOS {
	case "darwin":
		targetPath = filepath.Join(home, "Library/LaunchAgents/com.thismixer.mclip.plist")
		exec.Command("launchctl", "unload", targetPath).Run()
	case "linux":
		targetPath = filepath.Join(home, ".config/systemd/user/mclip.service")
		exec.Command("systemctl", "--user", "disable", "mclip", "--now").Run()
	}

	if err := os.Remove(targetPath); err != nil {
		fmt.Printf("[-] Не удалось удалить файл (возможно, его нет): %v\n", err)
	} else {
		fmt.Println("[+] Программа удалена из автозагрузки.")
	}
}

func showDevices() {
	cmd := exec.Command("pgrep", "-f", os.Args[0])
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("[-] MultiClip сейчас не запущен.")
	} else {
		fmt.Printf("[+] MultiClip работает (PID: %s). \n[!] Для просмотра активных соединений в реальном времени используйте: mclip -v\n", bytes.TrimSpace(output))
	}
}

func installService() {
	executable, _ := os.Executable()
	if filepath.Base(executable) == "main" || filepath.Ext(executable) == ".tmp" {
		fmt.Println("[!] Внимание: вы устанавливаете временный бинарник. Рекомендуется сначала сделать 'go build'.")
	}

	home, _ := os.UserHomeDir()
	data := struct{ Executable string }{Executable: executable}
	var targetPath string
	var tplContent string

	switch runtime.GOOS {
	case "darwin":
		targetPath = filepath.Join(home, "Library/LaunchAgents/com.thismixer.mclip.plist")
		tplContent = macTemplate
	case "linux":
		targetPath = filepath.Join(home, ".config/systemd/user/mclip.service")
		tplContent = linuxTemplate
	}

	tmpl, _ := template.New("install").Parse(tplContent)
	var buf bytes.Buffer
	tmpl.Execute(&buf, data)

	os.MkdirAll(filepath.Dir(targetPath), 0755)
	os.WriteFile(targetPath, buf.Bytes(), 0644)

	if runtime.GOOS == "darwin" {
		exec.Command("launchctl", "unload", targetPath).Run()
		exec.Command("launchctl", "load", targetPath).Run()
	} else {
		exec.Command("systemctl", "--user", "daemon-reload").Run()
		exec.Command("systemctl", "--user", "enable", "mclip", "--now").Run()
	}
	fmt.Printf("[+] Успешно установлено в %s\n", targetPath)
}

func runApp(isVerbose bool) {
	cb := clipboard.New()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var (
		lastText      string
		lastImageHash [16]byte
		mu            sync.Mutex
	)

	addPeer := func(addr string) {
		if _, loaded := activePeers.LoadOrStore(addr, time.Now()); !loaded {
			if isVerbose {
				fmt.Printf("[+] Подключено устройство: %s\n", addr)
			}
		}
	}

	go network.StartServer(cb, "8080", func(text string) {
		mu.Lock()
		lastText = text
		mu.Unlock()
		if isVerbose {
			fmt.Printf("<- Текст: %s\n", limitString(text, 50))
		}
	}, func(imgData []byte) {
		mu.Lock()
		lastImageHash = md5.Sum(imgData)
		mu.Unlock()
		if isVerbose {
			fmt.Println("<- Изображение получено")
		}
	}, addPeer)

	go network.Advertise(ctx, 8080)
	go network.Discover(ctx, addPeer)

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
					broadcast(&activePeers, "", currentImg, isVerbose)
				}
			}

			currentText, errText := cb.GetText()
			if errText == nil && currentText != lastText && currentText != "" {
				lastText = currentText
				broadcast(&activePeers, currentText, nil, isVerbose)
			}
			mu.Unlock()
		}
	}
}

func startDaemon() {
	cmd := exec.Command(os.Args[0], "-v")
	cmd.Env = append(os.Environ(), "MCLIP_DAEMON=1")
	cmd.Start()
	fmt.Printf("[+] MultiClip ушел в фон (PID: %d)\n", cmd.Process.Pid)
	os.Exit(0)
}

func stopDaemon() {
	exec.Command("pkill", "-f", os.Args[0]).Run()
	fmt.Println("[!] Все процессы MultiClip остановлены.")
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
			} else {
				err = network.SendImage(address, img)
			}
			if err != nil {
				remotes.Delete(address)
				if isVerbose {
					fmt.Printf("[-] Устройство %s отключилось\n", address)
				}
			}
		}(addr)
		return true
	})
}
