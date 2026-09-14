package ui

import (
	"context"
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

type fakeClock struct{ now domain.Timestamp }

func (c fakeClock) Now() domain.Timestamp { return c.now }

type fakeIDGen struct{ next int }

func (g *fakeIDGen) NewPromptID() domain.PromptID {
	g.next++
	return domain.PromptID("t" + string(rune('0'+g.next)))
}
func (g *fakeIDGen) NewSessionID() domain.SessionID { return "s1" }

type nullStore struct{}

func (nullStore) LoadSession(id domain.SessionID) (domain.Board, error) { return domain.Board{}, nil }
func (nullStore) SaveSession(board domain.Board) error                  { return nil }
func (nullStore) ListSessions() ([]domain.Session, error)               { return nil, nil }

// fakeHerdr lässt ResolveAndPromptWithPrefix immer erfolgreich "senden" —
// für Tests, die einen tatsächlich erfolgreichen Send-Effect brauchen (nicht
// nur den ErrNoPromptToSend-Fehlerpfad, der ohne HerdrGateway auskommt).
type fakeHerdr struct{}

func (fakeHerdr) CurrentPane(ctx context.Context) (ports.PaneInfo, error) {
	return ports.PaneInfo{ID: "pane-1"}, nil
}
func (fakeHerdr) NeighborPane(ctx context.Context, paneID string, direction ports.Direction) (ports.PaneInfo, error) {
	return ports.PaneInfo{ID: "pane-1"}, nil
}
func (fakeHerdr) AgentPrompt(ctx context.Context, paneID string, text string) error { return nil }
func (fakeHerdr) ResolveAndPromptWithPrefix(ctx context.Context, direction ports.Direction, prefixCommands []string, text string) error {
	return nil
}

func newTestModel() Model {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	board = board.AddPrompt("t2", "zweite Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, nil)
	return New(state, executor, fakeClock{now: 100}, &fakeIDGen{})
}

func TestBoardKey_Down_MovesFocusToNextPrompt(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	next := updated.(Model)

	if next.state.FocusedID == nil {
		t.Fatal("erwarte gesetzten Fokus")
	}
	if *next.state.FocusedID != domain.PromptID("t1") {
		t.Fatalf("erwarte Fokus auf t1 (erste Position), habe %s", *next.state.FocusedID)
	}
}

func TestBoardKey_ShiftJ_MovesFocusedPromptDown(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	updated, _ = updated.(Model).Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("J")})
	next := updated.(Model)

	live := next.state.Board.LivePrompts()
	if live[0].ID != domain.PromptID("t2") {
		t.Fatalf("erwarte t2 an erster Position nach J, habe %s", live[0].ID)
	}
	if live[1].ID != domain.PromptID("t1") {
		t.Fatalf("erwarte t1 an zweiter Position nach J, habe %s", live[1].ID)
	}
}

func TestBoardKey_G_FocusesLastPrompt_g_FocusesFirstPrompt(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "eins", 100)
	board = board.AddPrompt("t2", "zwei", 100)
	board = board.AddPrompt("t3", "drei", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, nil)
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	next := updated.(Model)
	if *next.state.FocusedID != domain.PromptID("t3") {
		t.Fatalf("erwarte Fokus auf t3 (letzter Prompt) nach G, habe %s", *next.state.FocusedID)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	next = updated.(Model)
	if *next.state.FocusedID != domain.PromptID("t1") {
		t.Fatalf("erwarte Fokus auf t1 (erster Prompt) nach g, habe %s", *next.state.FocusedID)
	}
}

func TestBoardKey_S_DispatchesSendSelectionUp(t *testing.T) {
	// Kein Prompt fokussiert/ausgewählt: Reduce liefert ErrNoPromptToSend,
	// ohne den (hier nil) HerdrGateway anzufassen — beweist, dass "s" bis in
	// den Reducer als SendSelectionToPane{DirectionUp} ankommt, ohne dass
	// ein echtes Gateway nötig ist.
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	next := updated.(Model)

	if !errors.Is(next.err, application.ErrNoPromptToSend) {
		t.Fatalf("erwarte ErrNoPromptToSend über 's', habe %v", next.err)
	}
}

func TestBoardKey_S_RemovesPromptAfterSuccessfulSend_ShiftUp_KeepsIt(t *testing.T) {
	newModelWithFakeHerdr := func() Model {
		board := domain.Board{}
		board = board.AddPrompt("t1", "erste Idee", 100)
		board = board.AddPrompt("t2", "zweite Idee", 100)
		state := application.AppState{Board: board, History: application.NewHistory(50)}
		executor := application.NewExecutor(nullStore{}, fakeHerdr{})
		return New(state, executor, fakeClock{now: 100}, &fakeIDGen{})
	}

	// "s": fokussierten Prompt senden -> nach Erfolg aus dem Board entfernt.
	m := newModelWithFakeHerdr()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("unerwarteter Fehler nach 's': %v", m.err)
	}
	live := m.state.Board.LivePrompts()
	if len(live) != 1 || live[0].ID != "t2" {
		t.Fatalf("erwarte nur noch t2 nach 's', habe %+v", live)
	}

	// Undo macht das Entfernen rückgängig.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("u")})
	m = updated.(Model)
	live = m.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte 2 Prompts nach Undo, habe %d", len(live))
	}

	// shift+up (nicht direkt über msg.String() simulierbar, daher als exakt
	// dieselbe Action dispatcht, die handleBoardKey für "shift+up" erzeugt):
	// derselbe Send, aber ohne RemoveAfterSend bleibt der Prompt in der Liste.
	m2 := newModelWithFakeHerdr()
	updated, _ = m2.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m2 = updated.(Model)
	m2 = m2.dispatch(application.SendSelectionToPane{Direction: ports.DirectionUp})

	if m2.err != nil {
		t.Fatalf("unerwarteter Fehler nach shift+up: %v", m2.err)
	}
	live = m2.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte weiterhin 2 Prompts nach shift+up (kein Entfernen), habe %d", len(live))
	}
}

func TestBoardKey_Space_TogglesMarkedOnFocusedPrompt(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if !m.state.Board.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked=true nach Leertaste")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = updated.(Model)
	if m.state.Board.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked=false nach zweiter Leertaste")
	}
}

func TestBoardKey_S_KeepsMarkedPromptAfterSend(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	board = board.AddPrompt("t2", "zweite Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, fakeHerdr{})
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // t1 markieren
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("unerwarteter Fehler nach 's' auf markierten Prompt: %v", m.err)
	}
	live := m.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte weiterhin 2 Prompts (markierter Prompt bleibt nach 's'), habe %d", len(live))
	}
	if !live[0].Marked {
		t.Fatal("erwarte t1 weiterhin markiert nach 's'")
	}
}

func TestBoardKey_ShiftS_SendsButKeepsPromptInList(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	board = board.AddPrompt("t2", "zweite Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, fakeHerdr{})
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("unerwarteter Fehler nach 'S': %v", m.err)
	}
	live := m.state.Board.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte weiterhin 2 Prompts nach 'S' (kein Entfernen), habe %d", len(live))
	}
}

func TestDispatch_SuccessfulSend_RecordsSentHistory(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee\nzweite Zeile", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, fakeHerdr{})
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("unerwarteter Fehler nach 's': %v", m.err)
	}
	history := m.state.Board.Session.SentHistory
	if len(history) != 1 || history[0].Text != "erste Idee zweite Zeile" {
		t.Fatalf("erwarte 1 SentHistory-Eintrag 'erste Idee zweite Zeile' (Zeilenumbruch durch Leerzeichen ersetzt), habe %+v", history)
	}
}

func TestDispatch_MultipleSends_AppendToSentHistory(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	board = board.AddPrompt("t2", "zweite Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, fakeHerdr{})
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")}) // sendet + entfernt t1
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown}) // t1 entfernt, Fokus war nil -> jetzt auf t2
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	history := m.state.Board.Session.SentHistory
	if len(history) != 2 || history[0].Text != "erste Idee" || history[1].Text != "zweite Idee" {
		t.Fatalf("erwarte 2 SentHistory-Einträge in Reihenfolge, habe %+v", history)
	}
}

func TestNew_PlanAndClearModeActiveByDefault(t *testing.T) {
	m := newTestModel()
	if !m.state.Board.Session.IsPlanModeActive() || !m.state.Board.Session.IsClearModeActive() {
		t.Fatalf("erwarte beide Modi per Default aktiv, habe plan=%v clear=%v",
			m.state.Board.Session.IsPlanModeActive(), m.state.Board.Session.IsClearModeActive())
	}
}

func TestBoardKey_P_TogglesPlanModePersistently(t *testing.T) {
	m := newTestModel()
	if !m.state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive initial true")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	if m.state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive false nach erstem 'p'")
	}

	// Persistent: eine andere Aktion dazwischen ändert PlanModeActive nicht.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive weiterhin false nach Navigation")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	if !m.state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive true nach zweitem 'p'")
	}
}

func TestBoardKey_C_TogglesClearModePersistently(t *testing.T) {
	m := newTestModel()
	if !m.state.Board.Session.IsClearModeActive() {
		t.Fatal("erwarte ClearModeActive initial true")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if m.state.Board.Session.IsClearModeActive() {
		t.Fatal("erwarte ClearModeActive false nach erstem 'c'")
	}

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	if !m.state.Board.Session.IsClearModeActive() {
		t.Fatal("erwarte ClearModeActive true nach zweitem 'c'")
	}
}

func TestBoardKey_P_ThenC_DeactivatesBothModes(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)

	if m.state.Board.Session.IsPlanModeActive() || m.state.Board.Session.IsClearModeActive() {
		t.Fatalf("erwarte beide Modi inaktiv nach 'p' dann 'c', habe plan=%v clear=%v",
			m.state.Board.Session.IsPlanModeActive(), m.state.Board.Session.IsClearModeActive())
	}
}

func TestBoardKey_C_ThenP_DeactivatesBothModes(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	m = updated.(Model)

	if m.state.Board.Session.IsPlanModeActive() || m.state.Board.Session.IsClearModeActive() {
		t.Fatalf("erwarte beide Modi inaktiv nach 'c' dann 'p', habe plan=%v clear=%v",
			m.state.Board.Session.IsPlanModeActive(), m.state.Board.Session.IsClearModeActive())
	}
}

func TestBoardKey_S_WithBothModesActive_SendsClearThenPlanPrefix(t *testing.T) {
	m := newTestModel()
	m.executor = application.NewExecutor(nullStore{}, fakeHerdr{})

	got := m.activePrefixCommands()
	want := []string{prefixCommandClear, prefixCommandPlan}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("erwarte Sendereihenfolge %v, habe %v", want, got)
	}
}

func TestBoardKey_S_WithPlanPrefix_SendsPrefixInAction(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	executor := application.NewExecutor(nullStore{}, fakeHerdr{})
	m := New(state, executor, fakeClock{now: 100}, &fakeIDGen{}) // Plan-Modus per Default an

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = updated.(Model)

	if m.err != nil {
		t.Fatalf("unerwarteter Fehler nach 's' mit Plan-Modus: %v", m.err)
	}
	history := m.state.Board.Session.SentHistory
	if len(history) != 1 || history[0].Text != "erste Idee" {
		t.Fatalf("erwarte SentHistory-Eintrag 'erste Idee', habe %+v", history)
	}
}

func TestBoardKey_H_OpensHelp_AnyKeyClosesIt(t *testing.T) {
	m := newTestModel()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	next := updated.(Model)
	if next.mode != modeHelp {
		t.Fatalf("erwarte modeHelp nach 'h', habe %v", next.mode)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	next = updated.(Model)
	if next.mode != modeBoard {
		t.Fatalf("erwarte modeBoard nach beliebiger Taste, habe %v", next.mode)
	}
}
