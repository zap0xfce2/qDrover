package herdr

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"time"

	"qdrover/internal/ports"
)

// prefixSettleDelay gibt der Ziel-Session Zeit, ein vorab gesendetes
// Präfix-Kommando (z. B. "/plan", "/clear") zu verarbeiten, bevor der
// eigentliche Text nachgeschickt wird. Als var statt const, damit Tests sie
// kurzzeitig verkleinern können (kein Sleep-Zwang).
var prefixSettleDelay = 1 * time.Second

type Runner interface {
	Run(ctx context.Context, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "herdr", args...)
	return cmd.Output()
}

type Gateway struct {
	runner Runner
}

func NewGateway() *Gateway {
	return &Gateway{runner: execRunner{}}
}

func NewGatewayWithRunner(r Runner) *Gateway {
	return &Gateway{runner: r}
}

func herdrEnabled() bool {
	return os.Getenv("HERDR_ENV") == "1" && os.Getenv("QDROVER_DISABLE_HERDR") == ""
}

type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// currentPaneResponse bildet die Antwort von "herdr pane current" ab:
// result.pane.pane_id (verifiziert gegen echte herdr-Ausgabe, 2026-09-11).
type currentPaneResponse struct {
	Result *struct {
		Pane struct {
			PaneID string `json:"pane_id"`
		} `json:"pane"`
	} `json:"result,omitempty"`
	Error *rpcError `json:"error,omitempty"`
}

// neighborPaneResponse bildet die Antwort von "herdr pane neighbor" ab:
// result.neighbor.neighbor_pane_id — eine andere Struktur als "pane current"
// (verifiziert gegen echte herdr-Ausgabe, 2026-09-11).
type neighborPaneResponse struct {
	Result *struct {
		Neighbor struct {
			NeighborPaneID string `json:"neighbor_pane_id"`
		} `json:"neighbor"`
	} `json:"result,omitempty"`
	Error *rpcError `json:"error,omitempty"`
}

func (g *Gateway) CurrentPane(ctx context.Context) (string, error) {
	if !herdrEnabled() {
		return "", fmt.Errorf("herdr ist deaktiviert (HERDR_ENV=1 setzen)")
	}
	out, err := g.runner.Run(ctx, "pane", "current")
	if err != nil {
		return "", fmt.Errorf("herdr pane current: %w", err)
	}
	var resp currentPaneResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("pane current parsen: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("herdr pane current: %s (%s)", resp.Error.Message, resp.Error.Code)
	}
	if resp.Result == nil || resp.Result.Pane.PaneID == "" {
		return "", fmt.Errorf("herdr pane current: leere pane-id in antwort")
	}
	return resp.Result.Pane.PaneID, nil
}

func (g *Gateway) NeighborPane(ctx context.Context, paneID string, direction ports.Direction) (string, error) {
	if !herdrEnabled() {
		return "", fmt.Errorf("herdr ist deaktiviert (HERDR_ENV=1 setzen)")
	}
	out, err := g.runner.Run(ctx, "pane", "neighbor", "--pane", paneID, "--direction", string(direction))
	if err != nil {
		return "", fmt.Errorf("herdr pane neighbor: %w", err)
	}
	var resp neighborPaneResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", fmt.Errorf("pane neighbor parsen: %w", err)
	}
	if resp.Error != nil {
		return "", fmt.Errorf("herdr pane neighbor: %s (%s)", resp.Error.Message, resp.Error.Code)
	}
	if resp.Result == nil || resp.Result.Neighbor.NeighborPaneID == "" {
		return "", fmt.Errorf("herdr pane neighbor: leere pane-id in antwort")
	}
	return resp.Result.Neighbor.NeighborPaneID, nil
}

func (g *Gateway) AgentPrompt(ctx context.Context, paneID string, text string) error {
	if !herdrEnabled() {
		return fmt.Errorf("herdr ist deaktiviert (HERDR_ENV=1 setzen)")
	}
	if _, err := g.runner.Run(ctx, "agent", "prompt", paneID, text); err != nil {
		return fmt.Errorf("herdr agent prompt: %w", err)
	}
	return nil
}

// resolveTargetPane ermittelt die Ziel-Pane: aktuelle Pane, optional deren
// Nachbar in direction. Gemeinsame Logik für ResolveAndPromptWithPrefix.
func (g *Gateway) resolveTargetPane(ctx context.Context, direction ports.Direction) (string, error) {
	current, err := g.CurrentPane(ctx)
	if err != nil {
		return "", fmt.Errorf("aktuelle pane ermitteln: %w", err)
	}
	if direction == "" {
		return current, nil
	}
	target, err := g.NeighborPane(ctx, current, direction)
	if err != nil {
		return "", fmt.Errorf("nachbar-pane ermitteln: %w", err)
	}
	return target, nil
}

// ResolveAndPromptWithPrefix ermittelt die Ziel-Pane einmal und sendet text
// dorthin. Nicht-leere prefixCommands werden davor in der übergebenen
// Reihenfolge gesendet (z. B. "/clear", dann "/plan"), jeweils gefolgt von
// prefixSettleDelay (bricht bei ctx-Timeout/-Cancel sofort ab), bevor der
// nächste Prefix bzw. text nachgeschickt wird. Gemeinsame Logik für den
// TUI-Executor und den CLI-Befehl "send".
func (g *Gateway) ResolveAndPromptWithPrefix(ctx context.Context, direction ports.Direction, prefixCommands []string, text string) error {
	target, err := g.resolveTargetPane(ctx, direction)
	if err != nil {
		return err
	}
	for _, prefix := range prefixCommands {
		if prefix == "" {
			continue
		}
		if err := g.AgentPrompt(ctx, target, prefix); err != nil {
			return fmt.Errorf("präfix-kommando senden: %w", err)
		}
		select {
		case <-time.After(prefixSettleDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return g.AgentPrompt(ctx, target, text)
}
