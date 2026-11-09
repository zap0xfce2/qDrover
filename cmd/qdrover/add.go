package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"qdrover/internal/adapters/sysclock"
	"qdrover/internal/adapters/uuidgen"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

const runDirName = "run"

// runAddCommand verdrahtet `qdrover add` mit echtem Dateisystem, Uhr und IDs.
func runAddCommand(args []string, stdin io.Reader, out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("arbeitsverzeichnis ermitteln: %w", err)
	}
	stateRoot, err := defaultStateDir()
	if err != nil {
		return err
	}
	return runAdd(args, stdin, out, stateRoot, cwd, sysclock.New(), uuidgen.New())
}

// runAdd legt einen lokalen Prompt in cwd/qdrover.json an (Duplikate werden
// übersprungen) und stößt laufende TUIs zum Abgleich an.
func runAdd(args []string, stdin io.Reader, out io.Writer, stateRoot, cwd string, clock ports.Clock, ids ports.IDGenerator) error {
	text, err := resolveQuery(args, stdin)
	if err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	board, store, err := bootBoard(stateRoot, cwd, clock, ids)
	if err != nil {
		return err
	}
	if hasLocalPrompt(board, text) {
		fmt.Fprintln(out, "bereits vorhanden:", text)
		return nil
	}
	board = board.AddScopedPrompt(ids.NewPromptID(), text, domain.ScopeLocal, clock.Now())
	if _, err := store.Save(board); err != nil {
		return fmt.Errorf("prompt speichern: %w", err)
	}
	fmt.Fprintln(out, "hinzugefügt:", text)
	notifyInstances(filepath.Join(stateRoot, runDirName))
	return nil
}

func hasLocalPrompt(board domain.Board, text string) bool {
	for _, p := range board.LivePrompts() {
		if p.IsLocal() && strings.TrimSpace(p.Content) == text {
			return true
		}
	}
	return false
}
