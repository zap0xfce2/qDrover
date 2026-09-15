package application

import (
	"fmt"

	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

func Reduce(state AppState, action Action, clock ports.Clock, ids ports.IDGenerator) (AppState, []Effect, error) {
	switch a := action.(type) {
	case CreatePrompt:
		state.History.Push(state.Board.Prompts)
		newID := ids.NewPromptID()
		state.Board = state.Board.AddPrompt(newID, a.Content, clock.Now())
		state.FocusedID = &newID
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case EditPrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		b, err := state.Board.EditPrompt(a.ID, a.Content, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case DeletePrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		b, err := state.Board.DeletePrompt(a.ID, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case RestorePrompt:
		b, err := state.Board.RestorePrompt(a.ID, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case MovePrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		b, err := state.Board.MovePrompt(a.ID, a.To, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case FocusPrompt:
		id := a.ID
		state.FocusedID = &id
		return state, nil, nil

	case ToggleMarked:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		b, err := state.Board.ToggleMarked(a.ID)
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case Undo:
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		prevPrompts, ok := state.History.Undo(state.Board.Prompts)
		if !ok {
			return state, nil, nil
		}
		state.Board.Prompts = prevPrompts
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case Redo:
		prevIndex := indexOfFocused(state.Board.LivePrompts(), state.FocusedID)
		nextPrompts, ok := state.History.Redo(state.Board.Prompts)
		if !ok {
			return state, nil, nil
		}
		state.Board.Prompts = nextPrompts
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case TogglePlanMode:
		active := !state.Board.Session.IsPlanModeActive()
		state.Board.Session.PlanModeActive = &active
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case ToggleClearMode:
		active := !state.Board.Session.IsClearModeActive()
		state.Board.Session.ClearModeActive = &active
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case RecordSentText:
		state.Board.Session = state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: clock.Now(), Text: a.Text})
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case SendSelectionToPane:
		text, ids, err := collectTextToSend(state)
		if err != nil {
			return state, nil, err
		}
		text = a.TextPrefix + text
		return state, []Effect{SendDispatch{Direction: a.Direction, Text: text, PromptIDs: ids, PrefixCommands: a.PrefixCommands}}, nil
	}
	return state, nil, fmt.Errorf("unbekannte Action %T", action)
}

// indexOfFocused liefert den Index von focusedID in live, oder -1 wenn nicht
// vorhanden bzw. focusedID nil ist.
func indexOfFocused(live []domain.Prompt, focusedID *domain.PromptID) int {
	if focusedID == nil {
		return -1
	}
	for i, t := range live {
		if t.ID == *focusedID {
			return i
		}
	}
	return -1
}

// reconcileFocus setzt FocusedID auf den Prompt, der vor der Mutation direkt
// über dem zuletzt fokussierten stand (prevIndex-1, gegen die neue Listenlänge
// geclampt), wenn FocusedID nach einer Board-Mutation auf keinen lebenden
// Prompt mehr zeigt (z.B. nach dem Löschen des fokussierten Prompts durch
// Senden). Bleibt das Board leer, wird FocusedID nil.
func reconcileFocus(state AppState, prevIndex int) AppState {
	if state.FocusedID == nil {
		return state
	}
	live := state.Board.LivePrompts()
	for _, t := range live {
		if t.ID == *state.FocusedID {
			return state
		}
	}
	if len(live) == 0 {
		state.FocusedID = nil
		return state
	}
	newIndex := prevIndex - 1
	if newIndex < 0 {
		newIndex = 0
	}
	if newIndex >= len(live) {
		newIndex = len(live) - 1
	}
	state.FocusedID = &live[newIndex].ID
	return state
}

var ErrNoPromptToSend = fmt.Errorf("kein Prompt ausgewählt oder fokussiert")

// collectTextToSend liefert den Inhalt des fokussierten Prompts. Navigieren
// zu einem Prompt (FocusPrompt) ist die einzige "Auswahl", die es gibt.
func collectTextToSend(state AppState) (string, []domain.PromptID, error) {
	if state.FocusedID == nil {
		return "", nil, ErrNoPromptToSend
	}
	for _, t := range state.Board.LivePrompts() {
		if t.ID == *state.FocusedID {
			return t.Content, []domain.PromptID{t.ID}, nil
		}
	}
	return "", nil, ErrNoPromptToSend
}
