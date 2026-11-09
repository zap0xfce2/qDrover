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
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		newID := ids.NewPromptID()
		state.Board = state.Board.AddScopedPrompt(newID, a.Content, a.Scope, clock.Now())
		if a.Marked {
			// Domain-Toggle statt Reduce: ein Undo-Schritt pro Aktion
			state.Board, _ = state.Board.ToggleMarked(newID)
		}
		if indexOfFocused(VisibleLive(state), &newID) >= 0 {
			state.FocusedID = &newID
		} else {
			state = reconcileFocus(state, prevIndex)
		}
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case EditPrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		b, err := state.Board.EditPrompt(a.ID, a.Content, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case DeletePrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		b, err := state.Board.DeletePrompt(a.ID, clock.Now())
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case RestorePrompt:
		b, err := state.Board.RestorePrompt(a.ID)
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case MovePrompt:
		state.History.Push(state.Board.Prompts)
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
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
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		b, err := state.Board.ToggleMarked(a.ID)
		if err != nil {
			return state, nil, err
		}
		state.Board = b
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case Undo:
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		prevPrompts, ok := state.History.Undo(state.Board.Prompts)
		if !ok {
			return state, nil, nil
		}
		state.Board.Prompts = prevPrompts
		state = reconcileFocus(state, prevIndex)
		return state, []Effect{PersistBoard{Board: state.Board}}, nil

	case Redo:
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
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

	case ApplyMerged:
		prevIndex := indexOfFocused(VisibleLive(state), state.FocusedID)
		state.History.Rebase(rebaseOnto(state.Board.Prompts, a.Board.Prompts))
		state.Board = mergeKeepingSoftDeleted(state.Board, a.Board)
		return reconcileFocus(state, prevIndex), nil, nil

	case SendSelectionToPane:
		text, id, err := collectTextToSend(state)
		if err != nil {
			return state, nil, err
		}
		return state, []Effect{SendDispatch{Direction: a.Direction, Text: text, PromptID: id, PrefixCommands: a.PrefixCommands}}, nil
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
	live := VisibleLive(state)
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
func collectTextToSend(state AppState) (string, domain.PromptID, error) {
	if state.FocusedID == nil {
		return "", "", ErrNoPromptToSend
	}
	for _, t := range VisibleLive(state) {
		if t.ID == *state.FocusedID {
			return t.Content, t.ID, nil
		}
	}
	return "", "", ErrNoPromptToSend
}

// VisibleLive liefert die lebenden, sichtbaren Prompts: globale vor lokalen,
// innerhalb nach Position.
func VisibleLive(state AppState) []domain.Prompt {
	var global, local []domain.Prompt
	for _, p := range state.Board.LivePrompts() {
		if state.Visible != nil && !state.Visible(p) {
			continue
		}
		if p.IsLocal() {
			local = append(local, p)
		} else {
			global = append(global, p)
		}
	}
	return append(global, local...)
}

// rebaseOnto überträgt die Änderungen von before nach merged auf einen
// History-Snapshot; die eigenen Abweichungen des Snapshots bleiben erhalten,
// ebenso seine soft-gelöschten Prompts (für Undo eigener Löschungen).
func rebaseOnto(before, merged []domain.Prompt) func([]domain.Prompt) []domain.Prompt {
	return func(snapshot []domain.Prompt) []domain.Prompt {
		rebased := domain.Board{Prompts: domain.MergePrompts(before, snapshot, merged)}
		return mergeKeepingSoftDeleted(domain.Board{Prompts: snapshot}, rebased).Prompts
	}
}

// mergeKeepingSoftDeleted behält soft-gelöschte Prompts des alten Boards, die
// im Merge fehlen, damit RestorePrompt nach fehlgeschlagenem Senden greift.
func mergeKeepingSoftDeleted(current, merged domain.Board) domain.Board {
	known := make(map[domain.PromptID]bool, len(merged.Prompts))
	for _, p := range merged.Prompts {
		known[p.ID] = true
	}
	prompts := append([]domain.Prompt{}, merged.Prompts...)
	for _, p := range current.Prompts {
		if !p.IsLive() && !known[p.ID] {
			prompts = append(prompts, p)
		}
	}
	merged.Prompts = prompts
	return merged
}
