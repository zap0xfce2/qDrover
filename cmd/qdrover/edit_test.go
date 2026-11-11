package main

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"qdrover/internal/domain"
)

// seqIDs vergibt fortlaufende IDs, damit mehrere Prompts pro Test unterscheidbar sind.
type seqIDs struct{ n *int }

func (s seqIDs) NewPromptID() domain.PromptID {
	*s.n++
	return domain.PromptID(fmt.Sprintf("p-%d", *s.n))
}

func addPrompts(t *testing.T, stateRoot, cwd string, texts ...string) {
	t.Helper()
	ids := seqIDs{n: new(int)}
	for _, text := range texts {
		if err := runAdd([]string{text}, strings.NewReader(""), &bytes.Buffer{}, stateRoot, cwd, stubClock{}, ids); err != nil {
			t.Fatal(err)
		}
	}
}

func contentOf(t *testing.T, stateRoot, cwd string, id domain.PromptID) string {
	t.Helper()
	for _, p := range loadBoardIn(t, stateRoot, cwd).LivePrompts() {
		if p.ID == id {
			return p.Content
		}
	}
	t.Fatalf("prompt %s nicht gefunden", id)
	return ""
}

func TestEdit_ByID_ReplacesContent(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	addPrompts(t, stateRoot, cwd, "alt")

	if err := runEdit([]string{"p-1", "neu"}, strings.NewReader(""), &bytes.Buffer{}, stateRoot, cwd, stubClock{}); err != nil {
		t.Fatal(err)
	}

	if got := contentOf(t, stateRoot, cwd, "p-1"); got != "neu" {
		t.Fatalf("erwarte neu, habe %q", got)
	}
}

func TestEdit_ByRoadmapNumber_ReadsStdin(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	addPrompts(t, stateRoot, cwd, "#1 Erstes\n\nText", "#12 Zwölftes\n\nText")

	if err := runEdit([]string{"#1"}, strings.NewReader("#1 Erstes neu\n\nmehr\n"), &bytes.Buffer{}, stateRoot, cwd, stubClock{}); err != nil {
		t.Fatal(err)
	}

	if got := contentOf(t, stateRoot, cwd, "p-1"); got != "#1 Erstes neu\n\nmehr" {
		t.Fatalf("unerwarteter Inhalt %q", got)
	}
}

func TestEdit_UnknownNumber_Fails(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	addPrompts(t, stateRoot, cwd, "#12 Zwölftes")

	err := runEdit([]string{"1", "neu"}, strings.NewReader(""), &bytes.Buffer{}, stateRoot, cwd, stubClock{})

	if err == nil {
		t.Fatal("erwarte Fehler, #12 darf nicht auf 1 passen")
	}
}

func TestResolveLocalPromptRef_AmbiguousNumber_Fails(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	addPrompts(t, stateRoot, cwd, "#3 Eins", "#3 Zwei")

	_, err := resolveLocalPromptRef(loadBoardIn(t, stateRoot, cwd), "3")

	if err == nil {
		t.Fatal("erwarte Fehler bei mehrdeutiger Nummer")
	}
}

func TestRemove_ByRoadmapNumber_DeletesPrompt(t *testing.T) {
	stateRoot, cwd := t.TempDir(), t.TempDir()
	addPrompts(t, stateRoot, cwd, "#3 Gestrichen")

	if err := runRemove([]string{"#3"}, &bytes.Buffer{}, stateRoot, cwd, stubClock{}); err != nil {
		t.Fatal(err)
	}

	if live := loadBoardIn(t, stateRoot, cwd).LivePrompts(); len(live) != 0 {
		t.Fatalf("erwarte keinen lebenden Prompt mehr, habe %+v", live)
	}
}
