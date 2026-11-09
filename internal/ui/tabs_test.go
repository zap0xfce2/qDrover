package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

var (
	keyLeft  = tea.KeyMsg{Type: tea.KeyLeft}
	keyRight = tea.KeyMsg{Type: tea.KeyRight}
)

// onGlobalTab schaltet auf den Global-Reiter, ohne den Fokus anzufassen —
// für Bestandstests, die unmarkierte Prompts ab Start navigierbar erwarten.
func onGlobalTab(m Model) Model {
	m.tab = tabGlobal
	m.state.Visible = tabFilter(tabGlobal)
	return m
}

// newMixedScopeModel: g1 global, g2 global+fixiert, l1 lokal+fixiert.
func newMixedScopeModel() Model {
	board := domain.Board{}
	board = board.AddScopedPrompt("g1", "globale Idee", domain.ScopeGlobal, 100)
	board = board.AddScopedPrompt("g2", "fixierte globale Idee", domain.ScopeGlobal, 100)
	board = board.AddScopedPrompt("l1", "fixierte lokale Idee", domain.ScopeLocal, 100)
	board, _ = board.ToggleMarked("g2")
	board, _ = board.ToggleMarked("l1")
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	return New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{})
}

func pressKey(m Model, msg tea.Msg) Model {
	updated, _ := m.Update(msg)
	return updated.(Model)
}

func runeKey(r string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(r)}
}

func TestTabs_StartsOnFixed(t *testing.T) {
	m := newMixedScopeModel()

	if *m.state.FocusedID != domain.PromptID("g2") {
		t.Fatalf("erwarte Start auf Fixiert mit Fokus g2, habe %s", *m.state.FocusedID)
	}
}

func TestTabs_RightArrowSwitchesToGlobal(t *testing.T) {
	m := pressKey(newMixedScopeModel(), keyRight)

	if m.tab != tabGlobal {
		t.Fatalf("erwarte Reiter Global, habe %d", m.tab)
	}
}

func TestTabs_LeftAtFirstTabStays(t *testing.T) {
	m := pressKey(newMixedScopeModel(), keyLeft)

	if m.tab != tabFixed {
		t.Fatalf("erwarte Reiter Fixiert, habe %d", m.tab)
	}
}

func TestTabs_NewPromptInLocalTabIsLocal(t *testing.T) {
	m := pressKey(pressKey(newMixedScopeModel(), keyRight), keyRight)

	m = pressKey(m, runeKey("i"))

	if p := application.VisibleLive(m.state); len(p) != 2 {
		t.Fatalf("erwarte neuen lokalen Prompt im Lokal-Reiter, habe %v", p)
	}
}

func TestTabs_NewPromptInFixedTabIsGlobalAndMarked(t *testing.T) {
	m := pressKey(newMixedScopeModel(), runeKey("i"))

	live := application.VisibleLive(m.state)
	if created := live[1]; created.IsLocal() || !created.Marked {
		t.Fatalf("erwarte neuen Prompt global und fixiert, habe %+v", created)
	}
}

func TestTabs_FixedShowsOriginPrefix(t *testing.T) {
	out := newMixedScopeModel().View()

	if !strings.Contains(out, "L fixierte lokale Idee") {
		t.Fatalf("erwarte Herkunftspräfix L im Fixiert-Reiter, habe:\n%s", out)
	}
}

func TestTabs_MoveInFixedIsNoop(t *testing.T) {
	m := pressKey(newMixedScopeModel(), runeKey("J"))

	if m.state.Board.LivePrompts()[1].ID != domain.PromptID("g2") {
		t.Fatalf("erwarte unveränderte Reihenfolge, habe %v", m.state.Board.LivePrompts())
	}
}

func TestTabs_ViewNeverExceedsHeightWithTabBar(t *testing.T) {
	board := domain.Board{}
	for _, id := range []domain.PromptID{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"} {
		board = board.AddPrompt(id, "Prompt "+string(id), 100)
	}
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{}, &fakeClipboard{})
	m = pressKey(pressKey(m, tea.WindowSizeMsg{Width: 80, Height: 8}), keyRight)

	if lines := strings.Count(m.View(), "\n") + 1; lines > m.height {
		t.Fatalf("erwarte höchstens %d Zeilen, habe %d", m.height, lines)
	}
}
