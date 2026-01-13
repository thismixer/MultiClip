package clipboard

import (
	"os/exec"
	"strings"
)

type linuxClipboard struct{}

func newPlatformClipboard() Clipboard {
	return &linuxClipboard{}
}

func (l *linuxClipboard) GetText() (string, error) {
	out, err := exec.Command("wl-paste", "--no-newline").Output()
	if err != nil {
		out, err = exec.Command("xclip", "-selection", "clipboard", "-o").Output()
	}

	return string(out), err
}

func (l *linuxClipboard) SetText(text string) error {
	cmd := exec.Command("wl-copy")
	if strings.Contains(text, "") {
		cmd = exec.Command("xclip", "-selection", "clipboard")
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
