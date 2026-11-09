package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/adapters/clipboard"
	"qdrover/internal/adapters/herdr"
	"qdrover/internal/adapters/jsonstore"
	"qdrover/internal/adapters/sysclock"
	"qdrover/internal/adapters/uuidgen"
	"qdrover/internal/application"
	"qdrover/internal/domain"
	"qdrover/internal/ui"
)

const (
	undoHistoryDepth = 50
	globalFileName   = "global.json"
	localFileName    = "qdrover.json"
)

func defaultStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home-verzeichnis ermitteln: %w", err)
	}
	return filepath.Join(home, ".local", "state", "qdrover"), nil
}

// bootBoard lädt global.json und cwd/qdrover.json. Fehlt global.json, startet
// es mit leerem Board; die Datei entsteht beim ersten Speichern.
func bootBoard(stateRoot, cwd string) (domain.Board, *jsonstore.ScopedStore, error) {
	store := jsonstore.NewScopedStore(filepath.Join(stateRoot, globalFileName), filepath.Join(cwd, localFileName))
	board, err := store.Load()
	if err != nil {
		return domain.Board{}, nil, fmt.Errorf("board laden: %w", err)
	}
	return board, store, nil
}

func runTUI() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("arbeitsverzeichnis ermitteln: %w", err)
	}
	stateRoot, err := defaultStateDir()
	if err != nil {
		return err
	}

	clock := sysclock.New()
	ids := uuidgen.New()
	clip := clipboard.New()

	board, store, err := bootBoard(stateRoot, cwd)
	if err != nil {
		return err
	}

	gateway := herdr.NewGateway()
	executor := application.NewExecutor(store, gateway)
	state := application.AppState{Board: board, History: application.NewHistory(undoHistoryDepth)}

	model := ui.New(state, executor, clock, ids, clip).WithVersion(version)
	p := tea.NewProgram(model, tea.WithAltScreen())
	// Live-Refresh ist optional: ohne Socket (z. B. Pfad > ~104 Zeichen) läuft die TUI ohne ihn weiter.
	if stopListening, err := listenForRefresh(filepath.Join(stateRoot, runDirName), os.Getpid(), func() { p.Send(ui.RefreshMsg{}) }); err == nil {
		defer stopListening()
	}
	_, err = p.Run()
	return err
}
