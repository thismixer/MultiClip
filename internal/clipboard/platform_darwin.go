package clipboard

import (
	"os/exec"
	"strings"
)

type darwinClipboard struct{}

func newPlatformClipboard() Clipboard {
	return &darwinClipboard{}
}

func (d *darwinClipboard) GetText() (string, error) {
	out, err := exec.Command("pbpaste").Output()
	return string(out), err
}

func (d *darwinClipboard) SetText(text string) error {
	cmd := exec.Command("pbcopy")
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
