package application

import (
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

type Action interface{ isAction() }

type CreatePrompt struct{ Content string }
type EditPrompt struct {
	ID      domain.PromptID
	Content string
}
type DeletePrompt struct{ ID domain.PromptID }

// RestorePrompt macht ein optimistisches DeletePrompt rückgängig, wenn der
// zugehörige Herdr-Send fehlgeschlagen ist (siehe Model.beginSend). Bewusst
// keine für den Nutzer sichtbare Undo-Aktion: löst keinen History.Push aus.
type RestorePrompt struct{ ID domain.PromptID }
type MovePrompt struct {
	ID domain.PromptID
	To domain.PromptPosition
}
type FocusPrompt struct{ ID domain.PromptID }
type ToggleMarked struct{ ID domain.PromptID }
type Undo struct{}
type Redo struct{}
type TogglePlanMode struct{}
type ToggleClearMode struct{}

// RecordSentText hängt Text als neuen Eintrag an Session.SentHistory an
// (siehe domain.Session.RecordSent) — dispatcht von Model.dispatch nach
// jedem erfolgreich ausgeführten SendDispatch-Effect.
type RecordSentText struct{ Text string }

func (CreatePrompt) isAction()    {}
func (EditPrompt) isAction()      {}
func (DeletePrompt) isAction()    {}
func (RestorePrompt) isAction()   {}
func (MovePrompt) isAction()      {}
func (FocusPrompt) isAction()     {}
func (ToggleMarked) isAction()    {}
func (Undo) isAction()            {}
func (Redo) isAction()            {}
func (TogglePlanMode) isAction()  {}
func (ToggleClearMode) isAction() {}
func (RecordSentText) isAction()  {}

type SendSelectionToPane struct {
	Direction ports.Direction
	// RemoveAfterSend: gesendete Prompts nach erfolgreichem Senden aus dem
	// Board entfernen (Soft-Delete, über Undo wiederherstellbar) statt sie
	// stehen zu lassen.
	RemoveAfterSend bool
	// PrefixCommands: werden in dieser Reihenfolge vor dem eigentlichen Text
	// an die Ziel-Pane gesendet (z. B. ["/clear", "/plan"]). Leer = kein
	// Präfix, normaler Send.
	PrefixCommands []string
}

func (SendSelectionToPane) isAction() {}
