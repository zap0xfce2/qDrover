package ports

type Clipboard interface {
	ReadAll() (string, error)
	WriteAll(text string) error
}
