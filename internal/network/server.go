package network

import (
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/thismixer/MultiClip/internal/clipboard"
)

func StartServer(cb clipboard.Clipboard, port string, onText func(string), onImg func([]byte), onPeerFound func(string)) error {

	extractIP := func(r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			if host == "::1" {
				host = "127.0.0.1"
			}
			peerAddr := fmt.Sprintf("%s:%s", host, port)
			onPeerFound(peerAddr)
		}
	}

	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		extractIP(r)
		body, _ := io.ReadAll(r.Body)
		text := string(body)

		_ = cb.SetText(text)
		onText(text)
	})

	http.HandleFunc("/sync-image", func(w http.ResponseWriter, r *http.Request) {
		extractIP(r)
		body, _ := io.ReadAll(r.Body)
		if len(body) > 0 {
			err := cb.SetImage(body)
			if err != nil {
				fmt.Printf("Ошибка записи картинки: %v\n", err)
			}
			onImg(body)
		}
	})

	fmt.Printf("Сервер запущен на порту %s\n", port)
	return http.ListenAndServe(":"+port, nil)
}
