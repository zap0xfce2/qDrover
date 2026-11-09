package ui

import (
	"testing"

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
