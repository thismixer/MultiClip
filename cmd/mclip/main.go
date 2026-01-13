package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/thismixer/MultiClip/internal/clipboard"
)

func main() {
	cb := clipboard.New()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	fmt.Println("MultiClip запущен")

	lastText, _ := cb.GetText()

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
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}
