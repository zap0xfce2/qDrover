package application

import (
	"testing"

	"qdrover/internal/domain"
)

type fakeStore struct {
	boards map[domain.SessionID]domain.Board
}

func newFakeStore() *fakeStore { return &fakeStore{boards: map[domain.SessionID]domain.Board{}} }

func (s *fakeStore) LoadSession(id domain.SessionID) (domain.Board, error) {
	return s.boards[id], nil
}

func (s *fakeStore) SaveSession(board domain.Board) error {
	s.boards[board.Session.ID] = board
	return nil
}

func (s *fakeStore) ListSessions() ([]domain.Session, error) {
	var sessions []domain.Session
	for _, b := range s.boards {
		sessions = append(sessions, b.Session)
	}
	return sessions, nil
}

func TestLoadOrCreateBoard_CreatesNewSessionWhenNoneMatchesCwd(t *testing.T) {
	store := newFakeStore()
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	board, err := LoadOrCreateBoard(store, "/tmp/project", clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if board.Session.OriginCwd != "/tmp/project" {
		t.Fatalf("erwarte OriginCwd '/tmp/project', habe %q", board.Session.OriginCwd)
	}
	if len(store.boards) != 1 {
		t.Fatalf("erwarte 1 gespeicherte Session, habe %d", len(store.boards))
	}
}

func TestLoadOrCreateBoard_ReusesExistingSessionForSameCwd(t *testing.T) {
	store := newFakeStore()
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	first, err := LoadOrCreateBoard(store, "/tmp/project", clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	second, err := LoadOrCreateBoard(store, "/tmp/project", clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if first.Session.ID != second.Session.ID {
		t.Fatalf("erwarte gleiche Session-ID, habe %q und %q", first.Session.ID, second.Session.ID)
	}
}
