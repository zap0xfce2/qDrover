package main

import (
	"os"
	"path/filepath"
	"testing"

	"qdrover/internal/domain"
)

type stubClock struct{}

func (stubClock) Now() domain.Timestamp { return 42 }

type stubIDs struct{}

func (stubIDs) NewPromptID() domain.PromptID { return "p-new" }

func TestBoot_NoGlobal_StartsWithEmptyBoard(t *testing.T) {
	board, _, err := bootBoard(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if live := board.LivePrompts(); len(live) != 0 {
		t.Fatalf("erwarte leeres Board, habe %+v", live)
	}
}

func TestBoot_NoGlobal_WritesNothing(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()

	if _, _, err := bootBoard(stateRoot, cwd); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(stateRoot, globalFileName)); !os.IsNotExist(err) {
		t.Fatalf("erwarte keine global.json vor dem ersten Speichern, Stat-Fehler: %v", err)
	}
}

func TestBoot_GlobalCreatedOnFirstSave(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	board, store, err := bootBoard(stateRoot, cwd)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Save(board.AddPrompt("p1", "erster", 1)); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(stateRoot, globalFileName)); err != nil {
		t.Fatalf("erwarte global.json nach dem ersten Speichern: %v", err)
	}
}

func TestBoot_CorruptGlobal_ErrorAndFileUntouched(t *testing.T) {
	stateRoot := t.TempDir()
	globalPath := filepath.Join(stateRoot, globalFileName)
	if err := os.WriteFile(globalPath, []byte("{kaputt"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := bootBoard(stateRoot, t.TempDir())

	raw, _ := os.ReadFile(globalPath)
	if err == nil || string(raw) != "{kaputt" {
		t.Fatalf("erwarte Fehler und unveränderte global.json, habe err=%v, Inhalt %q", err, raw)
	}
}
