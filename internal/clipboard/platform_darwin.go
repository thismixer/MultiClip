package clipboard

import (
	"os"
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

func (d *darwinClipboard) GetImage() ([]byte, error) {
	tiffFile := "/tmp/mclip_temp.tiff"
	pngFile := "/tmp/mclip_screen.png"

	script := "write (the clipboard as TIFF picture) to (open for access POSIX file \"" + tiffFile + "\" with write permission)"
	if err := exec.Command("osascript", "-e", script).Run(); err != nil {
		return nil, err
	}

	_ = exec.Command("sips", "-s", "format", "png", tiffFile, "--out", pngFile).Run()
	data, err := os.ReadFile(pngFile)

	_ = os.Remove(tiffFile)
	_ = os.Remove(pngFile)
	return data, err
}

func (d *darwinClipboard) SetImage(data []byte) error {
	tempFile := "/tmp/mclip_in.png"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return err
	}

	script := "set the clipboard to (read (POSIX file \"" + tempFile + "\") as «class PNGf»)"
	err := exec.Command("osascript", "-e", script).Run()

	_ = os.Remove(tempFile)
	return err
}
