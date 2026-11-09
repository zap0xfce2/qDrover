package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

// externalAddStore simuliert einen Prompt, den ein anderer Prozess (z. B. qdrover add)
// zwischenzeitlich in die Datei geschrieben hat.
type externalAddStore struct{}

func (externalAddStore) Save(board domain.Board) (domain.Board, error) {
	return board.AddScopedPrompt("extern", "von außen", domain.ScopeLocal, 100), nil
}

func TestUpdate_RefreshMsg_ShowsExternallyAddedPrompt(t *testing.T) {
	state := application.AppState{Board: domain.Board{}, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(externalAddStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{})

	updated, _ := m.Update(RefreshMsg{})

	if len(updated.(Model).state.Board.LivePrompts()) != 1 {
		t.Fatalf("erwarte den extern hinzugefügten Prompt nach RefreshMsg, habe %+v", updated.(Model).state.Board.LivePrompts())
	}
}

func TestBoardKey_r_RefreshesFromFiles(t *testing.T) {
	state := application.AppState{Board: domain.Board{}, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(externalAddStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if len(updated.(Model).state.Board.LivePrompts()) != 1 {
		t.Fatalf("erwarte den extern hinzugefügten Prompt nach r, habe %+v", updated.(Model).state.Board.LivePrompts())
	}
}

func TestBoardKey_ShiftR_Redoes(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("R")})

	if len(updated.(Model).state.Board.LivePrompts()) != 1 {
		t.Fatalf("erwarte nach d, u, R wieder nur einen Prompt (Löschen wiederholt), habe %+v", updated.(Model).state.Board.LivePrompts())
	}
}

func TestLogTab_r_RefreshesFromFiles(t *testing.T) {
	state := application.AppState{Board: domain.Board{}, History: application.NewHistory(50)}
	m := onLogTab(New(state, application.NewExecutor(externalAddStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{}))

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})

	if len(updated.(Model).state.Board.LivePrompts()) != 1 {
		t.Fatalf("erwarte Refresh auch im Log-Reiter, habe %+v", updated.(Model).state.Board.LivePrompts())
	}
}
