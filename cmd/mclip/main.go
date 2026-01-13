package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/thismixer/MultiClip/internal/clipboard"
	"github.com/thismixer/MultiClip/internal/network"
)

func main() {
	cb := clipboard.New()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	var remoteAddr string

	if len(os.Args) > 1 {
		remoteAddr = os.Args[1]
	} else {
		fmt.Print("Введи айпи второго устройства: ")
		fmt.Scanln(&remoteAddr)
	}

	fmt.Printf("MultiClip запущен. \nСвязь с: %s\n", remoteAddr)

	var lastText string

	go network.StartServer(cb, "8080", func(text string) {
		lastText = text
		fmt.Printf("Получено: %s\n", text)
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
				go network.SendText(remoteAddr, currentText)
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
