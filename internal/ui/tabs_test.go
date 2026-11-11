package ui

import (
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// onLogTab schaltet auf den Log-Reiter, ohne den Fokus anzufassen.
func onLogTab(m Model) Model {
	m.tab = tabLog
	m.state.Visible = tabFilter(tabLog)
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

func TestTabs_OrderIsAllGlobalLocalFixedLog(t *testing.T) {
	want := []string{"Alle", "Global", "Lokal", "Fixiert", "Log"}

	if !reflect.DeepEqual(tabLabels, want) {
		t.Fatalf("erwarte Reiter %v, habe %v", want, tabLabels)
	}
}

func TestTabs_RightArrowFromFixedSwitchesToLog(t *testing.T) {
	m := pressKey(newMixedScopeModel(), keyRight)

	if m.tab != tabLog {
		t.Fatalf("erwarte Reiter Log, habe %d", m.tab)
	}
}

func TestTabs_LeftAtFirstTabStays(t *testing.T) {
	m := pressKey(newMixedScopeModel().applyTab(tabAll), keyLeft)

	if m.tab != tabAll {
		t.Fatalf("erwarte Reiter Alle, habe %d", m.tab)
	}
}

func TestTabs_AllShowsEveryPrompt(t *testing.T) {
	m := newMixedScopeModel().applyTab(tabAll)

	if p := application.VisibleLive(m.state); len(p) != 3 {
		t.Fatalf("erwarte alle 3 Prompts im Reiter Alle, habe %v", p)
	}
}

func TestTabs_AllShowsOriginPrefix(t *testing.T) {
	out := newMixedScopeModel().applyTab(tabAll).View()

	if !strings.Contains(out, "G globale Idee") {
		t.Fatalf("erwarte Herkunftspräfix G im Reiter Alle, habe:\n%s", out)
	}
}

func TestTabs_NewPromptInAllTabIsGlobal(t *testing.T) {
	m := pressKey(newMixedScopeModel().applyTab(tabAll), runeKey("i"))

	if created := application.VisibleLive(m.state)[2]; created.IsLocal() || created.Marked {
		t.Fatalf("erwarte neuen Prompt global und nicht fixiert, habe %+v", created)
	}
}

func TestTabBar_WidthStableAcrossTabsAndCounts(t *testing.T) {
	few := newMixedScopeModel()
	many := newMixedScopeModel().applyTab(tabLog)
	for i := 0; i < 20; i++ {
		many.state.Board = many.state.Board.AddPrompt(domain.PromptID("x"+string(rune('a'+i))), "mehr", 100)
	}

	if lipgloss.Width(few.tabBar()) != lipgloss.Width(many.tabBar()) {
		t.Fatalf("erwarte gleich breite Reiterzeile, habe %q vs %q", few.tabBar(), many.tabBar())
	}
}

func TestTabs_NewPromptInLocalTabIsLocal(t *testing.T) {
	m := pressKey(newMixedScopeModel(), keyLeft)

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

func TestLogTab_EditSendDeleteMarkAreNoops(t *testing.T) {
	for _, k := range []tea.KeyMsg{
		{Type: tea.KeyEnter},
		{Type: tea.KeyRunes, Runes: []rune("s")},
		{Type: tea.KeyRunes, Runes: []rune("d")},
		{Type: tea.KeyRunes, Runes: []rune(" ")},
		{Type: tea.KeyRunes, Runes: []rune("i")},
		{Type: tea.KeyRunes, Runes: []rune("V")},
		{Type: tea.KeyRunes, Runes: []rune("C")},
		{Type: tea.KeyShiftUp},
		{Type: tea.KeyRunes, Runes: []rune("u")},
		{Type: tea.KeyRunes, Runes: []rune("R")},
	} {
		t.Run(k.String(), func(t *testing.T) {
			herdr := &fakeHerdr{}
			clip := &fakeClipboard{}
			m := onLogTab(newTestModel())
			m.clipboard = clip
			m.executor = application.NewExecutor(nullStore{}, herdr)
			before := m.state.Board

			after := runSend(t, m, k)

			if !reflect.DeepEqual(after.state.Board, before) || after.mode != modeBoard || herdr.calls != 0 || clip.writeCalls != 0 {
				t.Fatalf("erwarte No-Op im Log-Reiter für %q", k.String())
			}
		})
	}
}

func TestLogTab_UndoWithHistoryIsNoop(t *testing.T) {
	m := newMixedScopeModel()
	m = m.dispatch(application.CreatePrompt{Content: "neu"})
	m = onLogTab(m)
	before := m.state.Board

	m = pressKey(m, runeKey("u"))

	if !reflect.DeepEqual(m.state.Board, before) {
		t.Fatal("erwarte kein Undo im Log-Reiter")
	}
}

func TestTabs_FocusIsRestoredPerTab(t *testing.T) {
	m := newMixedScopeModel().applyTab(tabAll)
	m = pressKey(m, runeKey("j"))
	m = pressKey(pressKey(m, keyRight), keyLeft)

	if *m.state.FocusedID != domain.PromptID("g2") {
		t.Fatalf("erwarte wiederhergestellten Fokus g2, habe %s", *m.state.FocusedID)
	}
}

func TestTabs_DeletedFocusFallsBackToFirst(t *testing.T) {
	m := newMixedScopeModel().applyTab(tabAll)
	m = pressKey(m, runeKey("j"))
	m = m.applyTab(tabGlobal)
	m.state.Board, _ = m.state.Board.DeletePrompt("g2", 100)
	m = m.applyTab(tabAll)

	if *m.state.FocusedID != domain.PromptID("g1") {
		t.Fatalf("erwarte Rückfall auf g1, habe %s", *m.state.FocusedID)
	}
}
