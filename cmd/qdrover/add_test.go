package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"qdrover/internal/adapters/jsonstore"
)

func runAddIn(t *testing.T, stateRoot, cwd string, text string) string {
	t.Helper()
	var out bytes.Buffer
	if err := runAdd([]string{text}, strings.NewReader(""), &out, stateRoot, cwd, stubClock{}, stubIDs{}); err != nil {
		t.Fatal(err)
	}
	return out.String()
}

func TestAdd_WritesLocalPrompt(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()

	runAddIn(t, stateRoot, cwd, "neues Thema")

	board, err := jsonstore.NewScopedStore(filepath.Join(stateRoot, globalFileName), filepath.Join(cwd, localFileName)).Load()
	if err != nil || len(board.LivePrompts()) != 1 || !board.LivePrompts()[0].IsLocal() {
		t.Fatalf("erwarte genau einen lokalen Prompt, habe %+v (err %v)", board.LivePrompts(), err)
	}
}

func TestAdd_DuplicateContent_IsSkipped(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	runAddIn(t, stateRoot, cwd, "neues Thema")

	out := runAddIn(t, stateRoot, cwd, "  neues Thema ")

	if !strings.Contains(out, "bereits vorhanden") {
		t.Fatalf("erwarte Hinweis auf Duplikat, habe %q", out)
	}
}
