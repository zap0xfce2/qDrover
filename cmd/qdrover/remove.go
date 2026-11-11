package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"qdrover/internal/adapters/sysclock"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

// runRemoveCommand verdrahtet `qdrover remove` mit echtem Dateisystem und Uhr.
func runRemoveCommand(args []string, out io.Writer) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("arbeitsverzeichnis ermitteln: %w", err)
	}
	stateRoot, err := defaultStateDir()
	if err != nil {
		return err
	}
	return runRemove(args, out, stateRoot, cwd, sysclock.New())
}

// runRemove löscht einen lokalen Prompt per ID oder Roadmap-Nummer aus cwd/qdrover.json und stößt
// laufende TUIs zum Abgleich an. Globale Prompts sind bewusst ausgenommen.
func runRemove(args []string, out io.Writer, stateRoot, cwd string, clock ports.Clock) error {
	if len(args) != 1 {
		return errors.New("aufruf: qdrover remove <prompt-id|nummer>")
	}
	board, store, err := bootBoard(stateRoot, cwd)
	if err != nil {
		return err
	}
	id, err := resolveLocalPromptRef(board, args[0])
	if err != nil {
		return err
	}
	board, err = board.DeletePrompt(id, clock.Now())
	if err != nil {
		return err
	}
	if _, err := store.Save(board); err != nil {
		return fmt.Errorf("prompt löschen: %w", err)
	}
	fmt.Fprintln(out, "gelöscht:", id)
	notifyInstances(filepath.Join(stateRoot, runDirName))
	return nil
}

func hasLocalPromptID(board domain.Board, id domain.PromptID) bool {
	for _, p := range board.LivePrompts() {
		if p.IsLocal() && p.ID == id {
			return true
		}
	}
	return false
}
