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

const undoHistoryDepth = 50

// firstLivePromptID liefert die ID des ersten lebenden Prompts, damit beim
// Programmstart direkt ein bestehender Prompt fokussiert ist statt nichts.
func firstLivePromptID(board domain.Board) *domain.PromptID {
	live := board.LivePrompts()
	if len(live) == 0 {
		return nil
	}
	return &live[0].ID
}

func defaultStateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("home-verzeichnis ermitteln: %w", err)
	}
	return filepath.Join(home, ".local", "state", "qdrover", "sessions"), nil
}

func runTUI() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("arbeitsverzeichnis ermitteln: %w", err)
	}

	stateDir, err := defaultStateDir()
	if err != nil {
		return err
	}
	store, err := jsonstore.NewFileStore(stateDir)
	if err != nil {
		return err
	}

	clock := sysclock.New()
	ids := uuidgen.New()
	clip := clipboard.New()

	board, err := application.LoadOrCreateBoard(store, cwd, clock, ids)
	if err != nil {
		return fmt.Errorf("board laden: %w", err)
	}

	gateway := herdr.NewGateway()
	executor := application.NewExecutor(store, gateway)
	state := application.AppState{Board: board, History: application.NewHistory(undoHistoryDepth)}
	state.FocusedID = firstLivePromptID(board)

	model := ui.New(state, executor, clock, ids, clip).WithVersion(version)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
