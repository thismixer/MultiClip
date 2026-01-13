package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/thismixer/MultiClip/internal/clipboard"
	"github.com/thismixer/MultiClip/internal/network"
)

func main() {
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
			fmt.Printf("[+] Соединение установлено: %s\n", addr)
		}
	}

	go network.StartServer(cb, "8080", func(text string) {
		mu.Lock()
		lastText = text
		mu.Unlock()
		fmt.Printf("Получен текст: %s\n", limitString(text, 50))
	}, func(imgData []byte) {
		mu.Lock()
		lastImageHash = md5.Sum(imgData)
		mu.Unlock()
		fmt.Println("Получено изображение")
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
					fmt.Println("Отправка изображения...")
					broadcast(&remotes, "", currentImg)
					mu.Unlock()
					continue
				}
			}

			currentText, errText := cb.GetText()
			if errText == nil && currentText != lastText && currentText != "" {
				lastText = currentText
				fmt.Printf("Отправка текста: %s\n", limitString(currentText, 50))
				broadcast(&remotes, currentText, nil)
			}
			mu.Unlock()
		}
	}
}

func limitString(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n]) + "..."
	}
	return string(runes)
}

func broadcast(remotes *sync.Map, text string, img []byte) {
	remotes.Range(func(key, value any) bool {
		addr := key.(string)
		go func(address string) {
			if text != "" {
				_ = network.SendText(address, text)
			} else if len(img) > 0 {
				_ = network.SendImage(address, img)
			}
		}(addr)
		return true
	})
}
