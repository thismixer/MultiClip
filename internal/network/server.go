package network

import (
	"io"
	"net/http"

	"github.com/thismixer/MultiClip/internal/clipboard"
)

func StartServer(cb clipboard.Clipboard, port string, onReceive func(string)) error {
	http.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}

		text := string(body)
		cb.SetText(text)
		onReceive(text)
	})

	return http.ListenAndServe(":"+port, nil)
}
