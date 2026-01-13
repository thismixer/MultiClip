package main

import (
	"context"
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

	var lastText string
	var remotes sync.Map

	fmt.Println("MultiClip запущен (автопоиск)")

	go network.StartServer(cb, "8080", func(text string) {
		lastText = text
		fmt.Printf("Получено: %s\n", text)
	})

	go network.Advertise(ctx, 8080)

	go network.Discover(ctx, func(addr string) {
		if _, loaded := remotes.LoadOrStore(addr, true); !loaded {
			fmt.Printf("Найдено устройство: %s\n", addr)
		}
	})

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\nВыключение")
			return
		default:
			currentText, err := cb.GetText()
			if err == nil && currentText != lastText && currentText != "" {
				lastText = currentText
				fmt.Printf("Скопировано: %s\n", currentText)

				remotes.Range(func(key, value any) bool {
					addr := key.(string)
					go network.SendText(addr, currentText)
					return true
				})
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
