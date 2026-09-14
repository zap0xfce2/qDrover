package application

import (
	"fmt"

	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

func LoadOrCreateBoard(store ports.Store, cwd string, clock ports.Clock, ids ports.IDGenerator) (domain.Board, error) {
	sessions, err := store.ListSessions()
	if err != nil {
		return domain.Board{}, fmt.Errorf("sessions auflisten: %w", err)
	}
	for _, s := range sessions {
		if s.OriginCwd == cwd {
			return store.LoadSession(s.ID)
		}
	}

	now := clock.Now()
	session := domain.Session{
		ID:           ids.NewSessionID(),
		OriginCwd:    cwd,
		CreatedAt:    now,
		LastOpenedAt: now,
	}
	board := domain.Board{Session: session}
	if err := store.SaveSession(board); err != nil {
		return domain.Board{}, fmt.Errorf("neue session speichern: %w", err)
	}
	return board, nil
}
