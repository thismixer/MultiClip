package clipboard

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"time"
)

type linuxClipboard struct {
	isWayland bool
}

func newPlatformClipboard() Clipboard {
	isWayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
	if _, err := exec.LookPath("wl-copy"); err == nil {
		isWayland = true
	}
	return &linuxClipboard{isWayland: isWayland}
}

func (l *linuxClipboard) GetText() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if l.isWayland {
		cmd = exec.CommandContext(ctx, "wl-paste", "--type", "text/plain", "--no-newline")
	} else {
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-o")
	}

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (l *linuxClipboard) SetText(text string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if l.isWayland {
		cmd = exec.CommandContext(ctx, "wl-copy", "--type", "text/plain")
	} else {
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}

func (l *linuxClipboard) GetImage() ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if l.isWayland {
		checkCmd := exec.CommandContext(ctx, "wl-paste", "--list-types")
		typesOut, err := checkCmd.Output()
		if err != nil {
			return nil, err
		}

		typesStr := string(typesOut)
		if !strings.Contains(typesStr, "image/") {
			return nil, nil
		}

		data, err := exec.CommandContext(ctx, "wl-paste", "--type", "image/png").Output()
		if err != nil {
			return nil, err
		}
		return data, nil
	} else {
		return exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", "image/png", "-o").Output()
	}
}

func (l *linuxClipboard) SetImage(data []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if l.isWayland {
		cmd = exec.CommandContext(ctx, "wl-copy", "--type", "image/png")
	} else {
		cmd = exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", "image/png")
	}

	cmd.Stdin = bytes.NewReader(data)
	return cmd.Run()
}
