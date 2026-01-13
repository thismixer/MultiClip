package clipboard

type Clipboard interface {
	GetText() (string, error)
	SetText(text string) error
	GetImage() ([]byte, error)
	SetImage(data []byte) error
}

func New() Clipboard {
	return newPlatformClipboard()
}
