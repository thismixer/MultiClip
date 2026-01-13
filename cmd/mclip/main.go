package main

import (
	"fmt"
	"log"

	"github.com/thismixer/MultiClip/internal/clipboard"
)

func main() {
	cb := clipboard.New()

	text, err := cb.GetText()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("буфер: %s\n", text)

	err = cb.SetText("test")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("буфер изменен")
}
