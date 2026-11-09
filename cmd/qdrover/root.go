package main

import (
	"errors"
	"fmt"
	"io/fs"
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
	"qdrover/internal/ports"
	"qdrover/internal/ui"
)

const (
	undoHistoryDepth      = 50
	globalFileName        = "global.json"
	localFileName         = "qdrover.json"
	legacySessionsDirName = "sessions"
)

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
	return filepath.Join(home, ".local", "state", "qdrover"), nil
}

// bootBoard lädt das Board; fehlt global.json, werden einmalig die Alt-Sessions
// aus stateRoot/sessions migriert (die Alt-Dateien bleiben unangetastet).
func bootBoard(stateRoot, cwd string, clock ports.Clock, ids ports.IDGenerator) (domain.Board, *jsonstore.ScopedStore, error) {
	globalPath := filepath.Join(stateRoot, globalFileName)
	_, err := os.Stat(globalPath)
	if errors.Is(err, fs.ErrNotExist) {
		err = migrateLegacySessions(stateRoot, cwd, clock, ids)
	}
	if err != nil {
		return domain.Board{}, nil, err
	}
	store := jsonstore.NewScopedStore(globalPath, filepath.Join(cwd, localFileName))
	board, err := store.Load()
	if err != nil {
		return domain.Board{}, nil, fmt.Errorf("board laden: %w", err)
	}
	return board, store, nil
}

func migrateLegacySessions(stateRoot, cwd string, clock ports.Clock, ids ports.IDGenerator) error {
	boards, err := loadLegacyBoards(filepath.Join(stateRoot, legacySessionsDirName))
	if err != nil {
		return err
	}
	migrated := application.MigrateSessions(boards)
	if len(boards) == 0 {
		now := clock.Now()
		migrated.Session = domain.Session{ID: ids.NewSessionID(), OriginCwd: cwd, CreatedAt: now, LastOpenedAt: now}
	}
	// Eigener Store ohne cwd-Bezug: migrierte Prompts sind global, es entsteht keine qdrover.json.
	writer := jsonstore.NewScopedStore(filepath.Join(stateRoot, globalFileName), filepath.Join(stateRoot, localFileName))
	if _, err := writer.Save(migrated); err != nil {
		return fmt.Errorf("migration schreiben: %w", err)
	}
	return nil
}

func loadLegacyBoards(sessionsDir string) ([]domain.Board, error) {
	if _, err := os.Stat(sessionsDir); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	legacy, err := jsonstore.NewFileStore(sessionsDir)
	if err != nil {
		return nil, err
	}
	sessions, err := legacy.ListSessions()
	if err != nil {
		return nil, fmt.Errorf("alt-sessions auflisten: %w", err)
	}
	boards := make([]domain.Board, 0, len(sessions))
	for _, session := range sessions {
		board, err := legacy.LoadSession(session.ID)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, nil
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

	board, store, err := bootBoard(stateRoot, cwd, clock, ids)
	if err != nil {
		return err
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
