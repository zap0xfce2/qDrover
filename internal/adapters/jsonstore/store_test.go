package jsonstore

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"qdrover/internal/domain"
)

func TestFileStore_SaveThenLoad_RoundtripsBoard(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	board := domain.Board{
		Session: domain.Session{ID: "s1", OriginCwd: "/tmp/project", CreatedAt: 100, LastOpenedAt: 100},
	}
	board = board.AddPrompt("t1", "erste Idee", 100)

	if err := store.SaveSession(board); err != nil {
		t.Fatalf("SaveSession fehlgeschlagen: %v", err)
	}

	loaded, err := store.LoadSession("s1")
	if err != nil {
		t.Fatalf("LoadSession fehlgeschlagen: %v", err)
	}
	if len(loaded.Prompts) != 1 || loaded.Prompts[0].Content != "erste Idee" {
		t.Fatalf("erwarte 1 Prompt 'erste Idee', habe %+v", loaded.Prompts)
	}
}

func TestFileStore_ListSessions_ReturnsAllSavedSessions(t *testing.T) {
	store, err := NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	_ = store.SaveSession(domain.Board{Session: domain.Session{ID: "s1", OriginCwd: "/a"}})
	_ = store.SaveSession(domain.Board{Session: domain.Session{ID: "s2", OriginCwd: "/b"}})

	sessions, err := store.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions fehlgeschlagen: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("erwarte 2 Sessions, habe %d", len(sessions))
	}
}

func TestFileStore_ListSessions_WarnsOnStderrForBrokenSessionFile(t *testing.T) {
	dir := t.TempDir()
	store, err := NewFileStore(dir)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	_ = store.SaveSession(domain.Board{Session: domain.Session{ID: "s1", OriginCwd: "/a"}})
	if err := os.WriteFile(filepath.Join(dir, "kaputt.json"), []byte("{nicht valides json"), 0o644); err != nil {
		t.Fatalf("kaputte Session-Datei schreiben fehlgeschlagen: %v", err)
	}

	origStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe erstellen fehlgeschlagen: %v", err)
	}
	os.Stderr = w

	sessions, listErr := store.ListSessions()

	os.Stderr = origStderr
	_ = w.Close()
	stderrOutput, _ := io.ReadAll(r)

	if listErr != nil {
		t.Fatalf("erwarte weiterhin fehlerfreien Best-Effort-Aufruf, habe %v", listErr)
	}
	if len(sessions) != 1 {
		t.Fatalf("erwarte 1 gültige Session trotz kaputter Datei, habe %d", len(sessions))
	}
	if !strings.Contains(string(stderrOutput), "kaputt") {
		t.Fatalf("erwarte Warnung auf stderr für 'kaputt', habe %q", stderrOutput)
	}
}
