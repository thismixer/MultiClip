package network

import (
	"bytes"
	"net/http"
)

func SendText(addr string, text string) error {
	url := "http://" + addr + "/sync"
	_, err := http.Post(url, "text/plain", bytes.NewBufferString(text))
	return err
}
