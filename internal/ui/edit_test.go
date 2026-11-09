package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

func TestEditFlow_NewPrompt_TypeThenEscape_PersistsToNewPromptNotOldFocus(t *testing.T) {
	// eigene IDs (nicht "t1"/"t2"), damit sie nicht mit der vom fakeIDGen
	// erzeugten ID des neuen Prompts kollidieren.
	board := domain.Board{}
	board = board.AddPrompt("alt-a", "erste Idee", 100)
	board = board.AddPrompt("alt-b", "zweite Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf alt-a
	m = updated.(Model)
	oldFocusedID := *m.state.FocusedID
	oldContent := m.focusedContent()

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}) // "i" -> neuer Prompt
	m = updated.(Model)

	if m.state.FocusedID == nil || *m.state.FocusedID == oldFocusedID {
		t.Fatalf("erwarte Fokus auf neuen Prompt, habe %v (alt: %v)", m.state.FocusedID, oldFocusedID)
	}
	newFocusedID := *m.state.FocusedID

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("neu!")})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	live := m.state.Board.LivePrompts()
	if len(live) != 3 {
		t.Fatalf("erwarte 3 Prompts, habe %d", len(live))
	}
	for _, th := range live {
		if th.ID == oldFocusedID && th.Content != oldContent {
			t.Fatalf("alter fokussierter Prompt wurde überschrieben: %q", th.Content)
		}
		if th.ID == newFocusedID && th.Content != "neu!" {
			t.Fatalf("erwarte 'neu!' im neuen Prompt, habe %q", th.Content)
		}
	}
}

func TestEditFlow_NewPrompt_EmptyThenEscape_DeletesPrompt(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}) // "i" -> neuer, leerer Prompt
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc}) // sofort verlassen, ohne etwas einzugeben
	m = updated.(Model)

	live := m.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte 2 Prompts (neuer leerer Prompt wieder gelöscht), habe %d", len(live))
	}
}

func TestEditFlow_NewPrompt_WhitespaceOnlyThenEscape_DeletesPrompt(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("i")}) // "i" -> neuer, leerer Prompt
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")}) // nur Leerzeichen eingegeben
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	live := m.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte 2 Prompts (nur-Whitespace-Prompt wieder gelöscht), habe %d", len(live))
	}
}

func TestHandleEditKey_Enter_CommitsAndLeavesEditMode(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // in Edit-Modus wechseln
	m = updated.(Model)
	m.editor.SetValue("") // vorbefüllten Inhalt für den Test ignorieren

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	if m.mode != modeBoard {
		t.Fatalf("erwarte modeBoard nach Enter, habe %v", m.mode)
	}
	live := m.state.Board.LivePrompts()
	if live[0].Content != "!" {
		t.Fatalf("erwarte '!', habe %q", live[0].Content)
	}
}

func TestHandleEditKey_CtrlJ_InsertsNewline(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // in Edit-Modus wechseln
	m = updated.(Model)
	m.editor.SetValue("") // vorbefüllten Inhalt für den Test ignorieren

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyCtrlJ})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m = updated.(Model)

	if m.editor.Value() != "a\nb" {
		t.Fatalf("erwarte Editor-Inhalt 'a\\nb', habe %q", m.editor.Value())
	}
	if m.mode != modeEdit {
		t.Fatalf("erwarte weiterhin modeEdit nach Ctrl+J, habe %v", m.mode)
	}
}

func TestHandleEditKey_KeySpace_InsertsSpace(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // in Edit-Modus wechseln
	m = updated.(Model)
	m.editor.SetValue("") // vorbefüllten Inhalt für den Test ignorieren

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace, Runes: []rune(" ")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("b")})
	m = updated.(Model)

	if m.editor.Value() != "a b" {
		t.Fatalf("erwarte Editor-Inhalt 'a b', habe %q", m.editor.Value())
	}
}

func TestEditFlow_TypeThenEscape_PersistsContentToFocusedPrompt(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}) // in Edit-Modus wechseln (Board-Modus: "enter")
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("!")})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	m = updated.(Model)

	live := m.state.Board.LivePrompts()
	if live[0].Content != "erste Idee!" {
		t.Fatalf("erwarte 'erste Idee!', habe %q", live[0].Content)
	}
}

// editFocusedPrompt öffnet den Edit-Modus für t1 ("erste Idee"), Cursor am Ende.
func editFocusedPrompt(t *testing.T) Model {
	t.Helper()
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	return updated.(Model)
}

func sendKeys(m Model, keys ...tea.KeyMsg) Model {
	for _, k := range keys {
		updated, _ := m.Update(k)
		m = updated.(Model)
	}
	return m
}

func TestHandleEditKey_LeftArrow_InsertsBeforeCursor(t *testing.T) {
	m := editFocusedPrompt(t)

	m = sendKeys(m, tea.KeyMsg{Type: tea.KeyLeft}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	if m.editor.Value() != "erste IdeXe" {
		t.Fatalf("erwarte 'erste IdeXe', habe %q", m.editor.Value())
	}
}

func TestHandleEditKey_Home_InsertsAtLineStart(t *testing.T) {
	m := editFocusedPrompt(t)

	m = sendKeys(m, tea.KeyMsg{Type: tea.KeyHome}, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("X")})

	if m.editor.Value() != "Xerste Idee" {
		t.Fatalf("erwarte 'Xerste Idee', habe %q", m.editor.Value())
	}
}

func TestHandleEditKey_BackspaceAfterLeftArrow_DeletesMidText(t *testing.T) {
	m := editFocusedPrompt(t)

	m = sendKeys(m, tea.KeyMsg{Type: tea.KeyLeft}, tea.KeyMsg{Type: tea.KeyLeft}, tea.KeyMsg{Type: tea.KeyBackspace})

	if m.editor.Value() != "erste Iee" {
		t.Fatalf("erwarte 'erste Iee', habe %q", m.editor.Value())
	}
}

func TestHandleEditKey_LongContent_NotTruncated(t *testing.T) {
	m := editFocusedPrompt(t)
	long := strings.Repeat("x", 1000)

	m = sendKeys(m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(long)})

	if m.editor.Value() != "erste Idee"+long {
		t.Fatalf("erwarte ungekürzten Inhalt (%d Zeichen), habe %d", len("erste Idee"+long), len(m.editor.Value()))
	}
}
