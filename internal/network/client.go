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

func SendImage(addr string, imgData []byte) error {
	url := "http://" + addr + "/sync-image"
	resp, err := http.Post(url, "image/png", bytes.NewReader(imgData))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}
