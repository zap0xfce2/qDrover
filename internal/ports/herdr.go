package ports

import "context"

type Direction string

const (
	DirectionUp    Direction = "up"
	DirectionDown  Direction = "down"
	DirectionLeft  Direction = "left"
	DirectionRight Direction = "right"
)

type PaneInfo struct {
	ID string
}

type HerdrGateway interface {
	CurrentPane(ctx context.Context) (PaneInfo, error)
	NeighborPane(ctx context.Context, paneID string, direction Direction) (PaneInfo, error)
	AgentPrompt(ctx context.Context, paneID string, text string) error
	// ResolveAndPromptWithPrefix ermittelt die Ziel-Pane und sendet text
	// dorthin. prefixCommands werden davor in der übergebenen Reihenfolge
	// gesendet (z. B. "/clear", "/plan"), jeweils mit Settle-Delay danach,
	// an dieselbe Pane.
	ResolveAndPromptWithPrefix(ctx context.Context, direction Direction, prefixCommands []string, text string) error
}
