package ports

import "context"

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type HerdrGateway interface {
	// ResolveAndPromptWithPrefix ermittelt die Ziel-Pane und sendet text
	// dorthin. prefixCommands werden davor in der übergebenen Reihenfolge
	// gesendet (z. B. "/clear", "/plan"), jeweils mit Settle-Delay danach,
	// an dieselbe Pane.
	ResolveAndPromptWithPrefix(ctx context.Context, direction Direction, prefixCommands []string, text string) error
}
