package main

import (
	"os"
	"path/filepath"
	"testing"

	"qdrover/internal/adapters/jsonstore"
	"qdrover/internal/domain"
)

type stubClock struct{}

func (stubClock) Now() domain.Timestamp { return 42 }

type stubIDs struct{}

func (stubIDs) NewPromptID() domain.PromptID   { return "p-new" }
func (stubIDs) NewSessionID() domain.SessionID { return "s-new" }

func writeLegacySession(t *testing.T, stateRoot string, content string) {
	t.Helper()
	sessionsDir := filepath.Join(stateRoot, "sessions")
	legacy, err := jsonstore.NewFileStore(sessionsDir)
	if err != nil {
		t.Fatal(err)
	}
	board := domain.Board{Session: domain.Session{ID: "alt", OriginCwd: "/x", CreatedAt: 1, LastOpenedAt: 1}}
	board = board.AddPrompt("p1", content, 1)
	if err := legacy.SaveSession(board); err != nil {
		t.Fatal(err)
	}
}

func TestBoot_NoGlobal_MigratesLegacyPrompts(t *testing.T) {
	stateRoot := t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")

	board, _, err := bootBoard(stateRoot, t.TempDir(), stubClock{}, stubIDs{})
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if live := board.LivePrompts(); len(live) != 1 || live[0].Content != "alt-prompt" {
		t.Fatalf("erwarte migrierten Prompt, habe %v", live)
	}
}

// writeRawLegacySession schreibt eine Alt-Session-Datei, deren innere ID nicht zum Dateinamen passen muss.
func writeRawLegacySession(t *testing.T, stateRoot, fileName, innerID string) {
	t.Helper()
	sessionsDir := filepath.Join(stateRoot, "sessions")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"session": {"ID": "` + innerID + `"}, "prompts": []}`
	if err := os.WriteFile(filepath.Join(sessionsDir, fileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBoot_LegacySessionIDDiffersFromFileName_Skipped(t *testing.T) {
	stateRoot := t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")
	writeRawLegacySession(t, stateRoot, "kopie.json", "original")

	if _, _, err := bootBoard(stateRoot, t.TempDir(), stubClock{}, stubIDs{}); err != nil {
		t.Fatalf("Migration darf an fremder Session-ID nicht scheitern: %v", err)
	}
}

func TestBoot_LegacySessionWithEmptyID_Skipped(t *testing.T) {
	stateRoot := t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")
	writeRawLegacySession(t, stateRoot, "leer.json", "")

	if _, _, err := bootBoard(stateRoot, t.TempDir(), stubClock{}, stubIDs{}); err != nil {
		t.Fatalf("Migration darf an leerer Session-ID nicht scheitern: %v", err)
	}
}

func TestBoot_NoGlobal_WritesNoLocalFile(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")

	if _, _, err := bootBoard(stateRoot, cwd, stubClock{}, stubIDs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cwd, "qdrover.json")); !os.IsNotExist(err) {
		t.Fatalf("erwarte keine qdrover.json, Stat-Fehler: %v", err)
	}
}

func TestBoot_NoLegacySessions_WritesGlobalSoMigrationRunsOnce(t *testing.T) {
	stateRoot := t.TempDir()

	if _, _, err := bootBoard(stateRoot, t.TempDir(), stubClock{}, stubIDs{}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stateRoot, "global.json")); err != nil {
		t.Fatalf("erwarte global.json: %v", err)
	}
}

func TestBoot_ExistingGlobal_SkipsMigration(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")
	if _, _, err := bootBoard(stateRoot, cwd, stubClock{}, stubIDs{}); err != nil {
		t.Fatal(err)
	}
	writeLegacySession(t, stateRoot, "nachgereicht")

	board, _, err := bootBoard(stateRoot, cwd, stubClock{}, stubIDs{})
	if err != nil {
		t.Fatal(err)
	}
	if live := board.LivePrompts(); len(live) != 1 {
		t.Fatalf("erwarte weiterhin 1 Prompt, habe %d", len(live))
	}
}

func TestBoot_CorruptGlobal_NoMigrationAndError(t *testing.T) {
	stateRoot := t.TempDir()
	writeLegacySession(t, stateRoot, "alt-prompt")
	globalPath := filepath.Join(stateRoot, "global.json")
	if err := os.WriteFile(globalPath, []byte("{kaputt"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := bootBoard(stateRoot, t.TempDir(), stubClock{}, stubIDs{})
	if err == nil {
		t.Fatal("erwarte Fehler bei kaputter global.json")
	}
	raw, _ := os.ReadFile(globalPath)
	if string(raw) != "{kaputt" {
		t.Fatalf("global.json darf nicht überschrieben werden, habe %q", raw)
	}
}
