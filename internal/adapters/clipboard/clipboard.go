package clipboard

import (
	"fmt"
	"os"

	"github.com/atotto/clipboard"
	"github.com/aymanbagabas/go-osc52/v2"
)

type Clipboard struct{}

func New() Clipboard { return Clipboard{} }

func (Clipboard) ReadAll() (string, error) {
	text, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("zwischenablage lesen: %w", err)
	}
	return text, nil
}

// WriteAll schreibt sowohl per OSC52-Escape-Sequenz (funktioniert rein über
// die Terminalverbindung, auch über SSH, ganz ohne X11/Wayland) als auch —
// best effort — über das lokale Clipboard-Tool (xclip/xsel/wl-clipboard/
// pbcopy). Ein Fehlschlag des lokalen Tools (z. B. weil keine X11-/Wayland-
// Session erreichbar ist) wird bewusst nicht als Fehler zurückgegeben: der
// OSC52-Weg hat in dem Fall gute Chancen, trotzdem funktioniert zu haben, und
// lässt sich nicht separat verifizieren — ein Fehler hier wäre ein
// irreführender Fehlalarm in genau dem Terminal-only-Fall, für den OSC52
// gedacht ist.
func (Clipboard) WriteAll(text string) error {
	fmt.Fprint(os.Stdout, osc52.New(text).String())
	_ = clipboard.WriteAll(text)
	return nil
}
