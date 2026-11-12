package application

import (
	"testing"

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

func TestReduce_CreatePrompt_AddsPromptAndPersistEffect(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	newState, effects, err := Reduce(state, CreatePrompt{Content: "neue Idee"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := newState.Board.LivePrompts()
	if len(live) != 1 || live[0].Content != "neue Idee" {
		t.Fatalf("erwarte 1 Prompt 'neue Idee', habe %+v", live)
	}
	if len(effects) != 1 {
		t.Fatalf("erwarte 1 Effect, habe %d", len(effects))
	}
	if _, ok := effects[0].(PersistBoard); !ok {
		t.Fatalf("erwarte PersistBoard-Effect, habe %T", effects[0])
	}
}

func TestReduce_CreatePrompt_FocusesNewPromptNotOldOne(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "erster Prompt"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	oldID := state.Board.LivePrompts()[0].ID
	state.FocusedID = &oldID

	state, _, err = Reduce(state, CreatePrompt{Content: ""}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID == nil {
		t.Fatal("erwarte gesetzten Fokus nach CreatePrompt")
	}
	if *state.FocusedID == oldID {
		t.Fatal("erwarte Fokus auf neuen Prompt, nicht auf den alten")
	}
	newLive := state.Board.LivePrompts()
	if len(newLive) != 2 {
		t.Fatalf("erwarte 2 Prompts, habe %d", len(newLive))
	}

	state, _, err = Reduce(state, EditPrompt{ID: *state.FocusedID, Content: "neuer Inhalt"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	for _, th := range state.Board.LivePrompts() {
		if th.ID == oldID && th.Content != "erster Prompt" {
			t.Fatalf("alter Prompt wurde überschrieben: %q", th.Content)
		}
		if th.ID != oldID && th.Content != "neuer Inhalt" {
			t.Fatalf("neuer Prompt hat falschen Inhalt: %q", th.Content)
		}
	}
}

func TestReduce_SendSelectionToPane_UsesFocusedPrompt(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "zu sendende Idee"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	focusedID := state.Board.LivePrompts()[0].ID
	state.FocusedID = &focusedID

	_, effects, err := Reduce(state, SendSelectionToPane{Direction: ports.DirectionUp}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if len(effects) != 1 {
		t.Fatalf("erwarte 1 Effect, habe %d", len(effects))
	}
	sendEffect, ok := effects[0].(SendDispatch)
	if !ok {
		t.Fatalf("erwarte SendDispatch-Effect, habe %T", effects[0])
	}
	if sendEffect.Text != "zu sendende Idee" {
		t.Fatalf("erwarte Text 'zu sendende Idee', habe %q", sendEffect.Text)
	}
}

func TestReduce_SendSelectionToPane_FailsWithoutFocus(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	_, _, err := Reduce(state, SendSelectionToPane{Direction: ports.DirectionUp}, clock, ids)
	if err == nil {
		t.Fatal("erwarte Fehler ohne Fokus")
	}
}

func TestCollectTextToSend_FailsWhenFocusedPromptNoLongerExists(t *testing.T) {
	deadID := domain.PromptID("geloescht")
	state := AppState{
		Board:     domain.Board{},
		FocusedID: &deadID,
	}

	text, id, err := collectTextToSend(state)
	if err == nil {
		t.Fatal("erwarte Fehler, wenn der fokussierte Prompt nicht mehr lebt")
	}
	if text != "" {
		t.Fatalf("erwarte leeren Text bei Fehler, habe %q", text)
	}
	if id != "" {
		t.Fatalf("erwarte keine ID bei Fehler, habe %v", id)
	}
}

func TestReduce_ToggleMarked_TogglesPromptMarkedField(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "markieren"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	id := *state.FocusedID

	newState, effects, err := Reduce(state, ToggleMarked{ID: id}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !newState.Board.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked=true nach ToggleMarked")
	}
	if len(effects) != 1 {
		t.Fatalf("erwarte 1 Effect, habe %d", len(effects))
	}
	if _, ok := effects[0].(PersistBoard); !ok {
		t.Fatalf("erwarte PersistBoard-Effect, habe %T", effects[0])
	}
}

func TestReduce_ToggleMarked_FailsForUnknownPrompt(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	_, _, err := Reduce(state, ToggleMarked{ID: "unbekannt"}, clock, ids)
	if err == nil {
		t.Fatal("erwarte Fehler für unbekannte ID")
	}
}

func TestReduce_DeletePrompt_ClearsFocusForDeletedPrompt(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "wird gelöscht"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	id := *state.FocusedID

	state, _, err = Reduce(state, DeletePrompt{ID: id}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID != nil {
		t.Fatalf("erwarte FocusedID=nil nach Löschen des fokussierten Prompts, habe %v", *state.FocusedID)
	}
}

func TestReduce_DeletePrompt_FocusesFirstRemainingPromptWhenFocusedPromptDeleted(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "bleibt"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	firstID := *state.FocusedID

	state, _, err = Reduce(state, CreatePrompt{Content: "wird gelöscht"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	secondID := *state.FocusedID

	state, _, err = Reduce(state, DeletePrompt{ID: secondID}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID == nil || *state.FocusedID != firstID {
		t.Fatalf("erwarte FocusedID=%v (erster verbleibender Prompt), habe %v", firstID, state.FocusedID)
	}
}

func TestReduce_DeletePrompt_FocusesPromptAboveWhenMiddlePromptDeleted(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "erster"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	firstID := *state.FocusedID

	state, _, err = Reduce(state, CreatePrompt{Content: "mittlerer"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	middleID := *state.FocusedID

	state, _, err = Reduce(state, CreatePrompt{Content: "letzter"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	state, _, err = Reduce(state, FocusPrompt{ID: middleID}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	state, _, err = Reduce(state, DeletePrompt{ID: middleID}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID == nil || *state.FocusedID != firstID {
		t.Fatalf("erwarte FocusedID=%v (Prompt darüber), habe %v", firstID, *state.FocusedID)
	}
}

func TestReduce_DeletePrompt_FocusesPromptAboveWhenLastPromptDeleted(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "erster"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	state, _, err = Reduce(state, CreatePrompt{Content: "mittlerer"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	middleID := *state.FocusedID

	state, _, err = Reduce(state, CreatePrompt{Content: "letzter"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	lastID := *state.FocusedID

	state, _, err = Reduce(state, FocusPrompt{ID: lastID}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	state, _, err = Reduce(state, DeletePrompt{ID: lastID}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID == nil || *state.FocusedID != middleID {
		t.Fatalf("erwarte FocusedID=%v (Prompt darüber), habe %v", middleID, *state.FocusedID)
	}
}

func TestReduce_RestorePrompt_ClearsDeletedAt_NoHistoryPush(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "wird gelöscht und restauriert"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	id := *state.FocusedID

	state, _, err = Reduce(state, DeletePrompt{ID: id}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	historyDepthAfterDelete := len(state.History.past)

	state, effects, err := Reduce(state, RestorePrompt{ID: id}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := state.Board.LivePrompts()
	if len(live) != 1 || live[0].ID != id {
		t.Fatalf("erwarte Prompt %v wieder unter den lebenden Prompts, habe %+v", id, live)
	}
	if len(state.History.past) != historyDepthAfterDelete {
		t.Fatalf("erwarte unveränderte History-Tiefe (%d) nach RestorePrompt, habe %d", historyDepthAfterDelete, len(state.History.past))
	}
	if _, ok := effects[0].(PersistBoard); !ok {
		t.Fatalf("erwarte PersistBoard-Effect, habe %T", effects[0])
	}
}

func TestReduce_TogglePlanMode_TogglesSessionField(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, effects, err := Reduce(state, TogglePlanMode{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive=false nach erstem Toggle (Default war true)")
	}
	if len(effects) != 1 {
		t.Fatalf("erwarte 1 Effect, habe %d", len(effects))
	}
	if _, ok := effects[0].(PersistBoard); !ok {
		t.Fatalf("erwarte PersistBoard-Effect, habe %T", effects[0])
	}

	state, _, err = Reduce(state, TogglePlanMode{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive=true nach zweitem Toggle")
	}
}

func TestReduce_ToggleClearMode_TogglesSessionField(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, ToggleClearMode{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if state.Board.Session.IsClearModeActive() {
		t.Fatal("erwarte ClearModeActive=false nach erstem Toggle (Default war true)")
	}
}

func TestReduce_RecordSentText_AppendsToSessionSentHistory(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, effects, err := Reduce(state, RecordSentText{Text: "erste Idee"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	history := state.Board.Session.SentHistory
	if len(history) != 1 || history[0].Text != "erste Idee" || history[0].SentAt != 100 {
		t.Fatalf("erwarte 1 Eintrag ('erste Idee', SentAt=100), habe %+v", history)
	}
	if _, ok := effects[0].(PersistBoard); !ok {
		t.Fatalf("erwarte PersistBoard-Effect, habe %T", effects[0])
	}
}

func TestReduce_Undo_DoesNotRevertSessionFields(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "wird durch Undo entfernt"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	state, _, err = Reduce(state, TogglePlanMode{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	state, _, err = Reduce(state, RecordSentText{Text: "gesendet"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	state, _, err = Reduce(state, Undo{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if len(state.Board.LivePrompts()) != 0 {
		t.Fatalf("erwarte Prompt durch Undo entfernt, habe %d", len(state.Board.LivePrompts()))
	}
	if state.Board.Session.IsPlanModeActive() {
		t.Fatal("erwarte PlanModeActive weiterhin false — Undo darf Session nicht anrühren")
	}
	if len(state.Board.Session.SentHistory) != 1 {
		t.Fatalf("erwarte SentHistory weiterhin 1 Eintrag — Undo darf Session nicht anrühren, habe %d", len(state.Board.Session.SentHistory))
	}
}

func TestReduce_Undo_ClearsFocusForPromptThatNoLongerExistsAfterUndo(t *testing.T) {
	state := AppState{Board: domain.Board{}, History: NewHistory(50)}
	clock := fakeClock{now: 100}
	ids := &fakeIDGen{}

	state, _, err := Reduce(state, CreatePrompt{Content: "wird durch Undo entfernt"}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	// FocusedID zeigt bereits auf den neuen Prompt (C1); Undo entfernt ihn wieder.

	state, _, err = Reduce(state, Undo{}, clock, ids)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	if state.FocusedID != nil {
		t.Fatalf("erwarte FocusedID=nil nach Undo des CreatePrompt, habe %v", *state.FocusedID)
	}
}

func scopedFixtureBoard() domain.Board {
	return domain.Board{Prompts: []domain.Prompt{
		{ID: "g1", Position: 0},
		{ID: "l1", Position: 1, Scope: domain.ScopeLocal},
		{ID: "g2", Position: 2},
	}}
}

func TestReduce_CreatePrompt_UsesScopeAndMarked(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	state, _, _ = Reduce(state, CreatePrompt{Content: "x", Scope: domain.ScopeLocal, Marked: true}, fakeClock{now: 100}, &fakeIDGen{})

	got := state.Board.LivePrompts()[0]
	if !got.IsLocal() || !got.Marked {
		t.Fatalf("erwarte lokal+markiert, habe %+v", got)
	}
}

func TestReduce_CreatePrompt_MarkedPushesSingleHistoryEntry(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	state, _, _ = Reduce(state, CreatePrompt{Content: "x", Marked: true}, fakeClock{now: 100}, &fakeIDGen{})

	if len(state.History.past) != 1 {
		t.Fatalf("erwarte 1 History-Eintrag, habe %d", len(state.History.past))
	}
}

func TestReduce_DeleteFocused_FocusStaysInVisibleSet(t *testing.T) {
	focus := domain.PromptID("g2")
	state := AppState{
		Board: scopedFixtureBoard(), FocusedID: &focus, History: NewHistory(50),
		Visible: func(p domain.Prompt) bool { return !p.IsLocal() },
	}
	state, _, _ = Reduce(state, DeletePrompt{ID: "g2"}, fakeClock{now: 100}, &fakeIDGen{})

	if state.FocusedID == nil || *state.FocusedID != "g1" {
		t.Fatalf("erwarte Fokus g1, habe %v", state.FocusedID)
	}
}

func TestReduce_ToggleMarked_LastVisibleUnmarked_FocusNil(t *testing.T) {
	focus := domain.PromptID("a")
	state := AppState{
		Board:     domain.Board{Prompts: []domain.Prompt{{ID: "a", Marked: true}, {ID: "b", Position: 1}}},
		FocusedID: &focus, History: NewHistory(50),
		Visible: func(p domain.Prompt) bool { return p.Marked },
	}
	state, _, _ = Reduce(state, ToggleMarked{ID: "a"}, fakeClock{now: 100}, &fakeIDGen{})

	if state.FocusedID != nil {
		t.Fatalf("erwarte FocusedID=nil, habe %v", *state.FocusedID)
	}
}

func TestReduce_SendSelection_IgnoresFocusOutsideVisible(t *testing.T) {
	focus := domain.PromptID("l1")
	state := AppState{
		Board: scopedFixtureBoard(), FocusedID: &focus,
		Visible: func(p domain.Prompt) bool { return !p.IsLocal() },
	}
	_, _, err := Reduce(state, SendSelectionToPane{}, fakeClock{}, &fakeIDGen{})

	if err != ErrNoPromptToSend {
		t.Fatalf("erwarte ErrNoPromptToSend, habe %v", err)
	}
}

func TestVisibleLive_GlobalBeforeLocal(t *testing.T) {
	got := VisibleLive(AppState{Board: scopedFixtureBoard()})

	if got[0].ID != "g1" || got[1].ID != "g2" || got[2].ID != "l1" {
		t.Fatalf("erwarte g1,g2,l1, habe %+v", got)
	}
}

func TestReduce_ApplyMerged_KeepsSoftDeletedForRestore(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	clock, ids := fakeClock{now: 100}, &fakeIDGen{}
	state, _, _ = Reduce(state, CreatePrompt{Content: "x"}, clock, ids)
	id := *state.FocusedID
	state, _, _ = Reduce(state, DeletePrompt{ID: id}, clock, ids)

	state, _, _ = Reduce(state, ApplyMerged{Board: domain.Board{}}, clock, ids)
	state, _, err := Reduce(state, RestorePrompt{ID: id}, clock, ids)

	if err != nil || len(state.Board.LivePrompts()) != 1 {
		t.Fatalf("erwarte erfolgreiches Restore, err=%v prompts=%+v", err, state.Board.Prompts)
	}
}

func TestReduce_CreatePrompt_InvisibleNewPrompt_FocusStaysVisible(t *testing.T) {
	focus := domain.PromptID("g1")
	state := AppState{
		Board: scopedFixtureBoard(), FocusedID: &focus, History: NewHistory(50),
		Visible: func(p domain.Prompt) bool { return !p.IsLocal() },
	}
	state, _, _ = Reduce(state, CreatePrompt{Content: "x", Scope: domain.ScopeLocal}, fakeClock{now: 100}, &fakeIDGen{})

	if state.FocusedID == nil || *state.FocusedID != "g1" {
		t.Fatalf("erwarte Fokus g1, habe %v", state.FocusedID)
	}
}

// mergedWithForeign liefert das aktuelle Board plus einen fremden Prompt, so wie es ein Save-Merge zurückgäbe.
func mergedWithForeign(board domain.Board) domain.Board {
	return board.AddScopedPrompt("foreign", "von B", domain.ScopeGlobal, 50)
}

func TestReduce_UndoAfterApplyMerged_KeepsForeignPrompt(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	clock, ids := fakeClock{now: 100}, &fakeIDGen{}
	state, _, _ = Reduce(state, CreatePrompt{Content: "mine"}, clock, ids)
	state, _, _ = Reduce(state, ApplyMerged{Board: mergedWithForeign(state.Board)}, clock, ids)

	state, _, _ = Reduce(state, Undo{}, clock, ids)

	if got := state.Board.LivePrompts(); len(got) != 1 || got[0].ID != "foreign" {
		t.Fatalf("erwarte nur foreign, habe %+v", got)
	}
}

func TestReduce_UndoAfterApplyMerged_KeepsForeignEdit(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	clock, ids := fakeClock{now: 100}, &fakeIDGen{}
	state, _, _ = Reduce(state, CreatePrompt{Content: "a"}, clock, ids)
	first := *state.FocusedID
	state, _, _ = Reduce(state, CreatePrompt{Content: "b"}, clock, ids)
	foreignEdit, _ := state.Board.EditPrompt(first, "fremd", 99)
	state, _, _ = Reduce(state, ApplyMerged{Board: foreignEdit}, clock, ids)

	state, _, _ = Reduce(state, Undo{}, clock, ids)

	if got := state.Board.LivePrompts()[0].Content; got != "fremd" {
		t.Fatalf("erwarte fremd, habe %q", got)
	}
}

func TestReduce_RedoAfterApplyMerged_KeepsForeignPrompt(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	clock, ids := fakeClock{now: 100}, &fakeIDGen{}
	state, _, _ = Reduce(state, CreatePrompt{Content: "mine"}, clock, ids)
	state, _, _ = Reduce(state, Undo{}, clock, ids)
	state, _, _ = Reduce(state, ApplyMerged{Board: mergedWithForeign(state.Board)}, clock, ids)

	state, _, _ = Reduce(state, Redo{}, clock, ids)

	if got := len(state.Board.LivePrompts()); got != 2 {
		t.Fatalf("erwarte mine+foreign, habe %+v", state.Board.LivePrompts())
	}
}

func TestReduce_UndoOwnDeleteAfterApplyMerged_RestoresPrompt(t *testing.T) {
	state := AppState{History: NewHistory(50)}
	clock, ids := fakeClock{now: 100}, &fakeIDGen{}
	state, _, _ = Reduce(state, CreatePrompt{Content: "x"}, clock, ids)
	state, _, _ = Reduce(state, DeletePrompt{ID: *state.FocusedID}, clock, ids)
	state, _, _ = Reduce(state, ApplyMerged{Board: mergedWithForeign(domain.Board{})}, clock, ids)

	state, _, _ = Reduce(state, Undo{}, clock, ids)

	if got := len(state.Board.LivePrompts()); got != 2 {
		t.Fatalf("erwarte x+foreign, habe %+v", state.Board.LivePrompts())
	}
}

func TestReduce_Refresh_PersistsCurrentBoardWithoutHistory(t *testing.T) {
	board := domain.Board{}.AddPrompt("p1", "idee", 1)
	state := AppState{Board: board, History: NewHistory(50)}

	next, effects, _ := Reduce(state, Refresh{}, fakeClock{now: 1}, &fakeIDGen{})

	persist, ok := effects[0].(PersistBoard)
	_, undoable := next.History.Undo(nil)
	if !ok || len(persist.Board.Prompts) != 1 || undoable {
		t.Fatalf("erwarte PersistBoard mit aktuellem Board und keinen Undo-Eintrag, habe %+v", effects)
	}
}

func TestReduce_SetScope_MovesPromptToLocal(t *testing.T) {
	state := AppState{Board: domain.Board{}.AddPrompt("t1", "a", 100), History: NewHistory(50)}

	newState, _, _ := Reduce(state, SetScope{ID: "t1", Scope: domain.ScopeLocal}, fakeClock{now: 100}, &fakeIDGen{})

	if !newState.Board.LivePrompts()[0].IsLocal() {
		t.Fatal("erwarte lokalen Prompt nach SetScope")
	}
}

func TestReduce_SetScope_IsUndoable(t *testing.T) {
	state := AppState{Board: domain.Board{}.AddPrompt("t1", "a", 100), History: NewHistory(50)}
	state, _, _ = Reduce(state, SetScope{ID: "t1", Scope: domain.ScopeLocal}, fakeClock{now: 100}, &fakeIDGen{})

	state, _, _ = Reduce(state, Undo{}, fakeClock{now: 100}, &fakeIDGen{})

	if state.Board.LivePrompts()[0].IsLocal() {
		t.Fatal("erwarte globalen Prompt nach Undo")
	}
}
