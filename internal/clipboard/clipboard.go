package clipboard

type Clipboard interface {
	GetText() (string, error)
	SetText(text string) error
}

func New() Clipboard {
	return newPlatformClipboard()
}
