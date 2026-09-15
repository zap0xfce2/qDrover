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

func (g *fakeIDGen) NewSessionID() domain.SessionID { return "s1" }

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

	text, ids, err := collectTextToSend(state)
	if err == nil {
		t.Fatal("erwarte Fehler, wenn der fokussierte Prompt nicht mehr lebt")
	}
	if text != "" {
		t.Fatalf("erwarte leeren Text bei Fehler, habe %q", text)
	}
	if ids != nil {
		t.Fatalf("erwarte keine IDs bei Fehler, habe %v", ids)
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
