package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"qdrover/internal/adapters/jsonstore"
	"qdrover/internal/domain"
)

func loadBoardIn(t *testing.T, stateRoot, cwd string) domain.Board {
	t.Helper()
	board, err := jsonstore.NewScopedStore(filepath.Join(stateRoot, globalFileName), filepath.Join(cwd, localFileName)).Load()
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func TestRemove_DeletesLocalPrompt(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	runAddIn(t, stateRoot, cwd, "gestrichenes Thema")
	id := loadBoardIn(t, stateRoot, cwd).LivePrompts()[0].ID

	if err := runRemove([]string{string(id)}, &bytes.Buffer{}, stateRoot, cwd, stubClock{}); err != nil {
		t.Fatal(err)
	}

	if live := loadBoardIn(t, stateRoot, cwd).LivePrompts(); len(live) != 0 {
		t.Fatalf("erwarte keinen lebenden Prompt mehr, habe %+v", live)
	}
}

func TestRemove_UnknownID_Fails(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()

	err := runRemove([]string{"gibt-es-nicht"}, &bytes.Buffer{}, stateRoot, cwd, stubClock{})

	if err == nil {
		t.Fatal("erwarte Fehler für unbekannte ID")
	}
}
