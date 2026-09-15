package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

func (m Model) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "h":
		m.mode = modeHelp
		return m, nil
	case "p":
		return m.dispatch(application.TogglePlanMode{}), nil
	case "c":
		return m.dispatch(application.ToggleClearMode{}), nil
	case "down", "j":
		return m.moveFocus(1), nil
	case "up", "k":
		return m.moveFocus(-1), nil
	case "enter":
		m.mode = modeEdit
		m.editBuf = m.focusedContent()
		m.editingNewPrompt = false
		return m, nil
	case "i":
		m = m.dispatch(application.CreatePrompt{Content: ""})
		m.mode = modeEdit
		m.editBuf = ""
		m.editingNewPrompt = true
		return m, nil
	case "d":
		if m.state.FocusedID != nil {
			m = m.dispatch(application.DeletePrompt{ID: *m.state.FocusedID})
		}
		return m, nil
	case "u":
		return m.dispatch(application.Undo{}), nil
	case "r":
		return m.dispatch(application.Redo{}), nil
	case "s":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionUp, RemoveAfterSend: true, PrefixCommands: m.activePrefixCommands()})
	case "S":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionUp, RemoveAfterSend: true, TextPrefix: subtaskPrefix})
	case "shift+up":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionUp, PrefixCommands: m.activePrefixCommands()})
	case "shift+down":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionDown, PrefixCommands: m.activePrefixCommands()})
	case "shift+left":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionLeft, PrefixCommands: m.activePrefixCommands()})
	case "shift+right":
		return m.beginSend(application.SendSelectionToPane{Direction: ports.DirectionRight, PrefixCommands: m.activePrefixCommands()})
	case "J":
		return m.movePrompt(1), nil
	case "K":
		return m.movePrompt(-1), nil
	case "g":
		return m.focusFirst(), nil
	case "G":
		return m.focusLast(), nil
	case " ":
		return m.toggleMarked(), nil
	case "V":
		content, err := m.clipboard.ReadAll()
		if err != nil {
			m.err = err
			return m, nil
		}
		m = m.dispatch(application.CreatePrompt{Content: ""})
		m.mode = modeEdit
		m.editBuf = content
		m.editingNewPrompt = true
		return m, nil
	case "C":
		if m.state.FocusedID == nil {
			return m, nil
		}
		if err := m.clipboard.WriteAll(m.focusedContent()); err != nil {
			m.err = err
			return m, nil
		}
		m.err = nil
		return m, nil
	}
	return m, nil
}

// movePrompt verschiebt den fokussierten Prompt um delta Positionen (geclamped).
func (m Model) movePrompt(delta int) Model {
	if m.state.FocusedID == nil {
		return m
	}
	live := m.state.Board.LivePrompts()
	idx := focusedIndex(live, m.state.FocusedID)
	newPos := idx + delta
	if newPos < 0 {
		newPos = 0
	}
	if newPos > len(live)-1 {
		newPos = len(live) - 1
	}
	return m.dispatch(application.MovePrompt{ID: *m.state.FocusedID, To: domain.PromptPosition(newPos)})
}

func (m Model) focusFirst() Model {
	live := m.state.Board.LivePrompts()
	if len(live) == 0 {
		return m
	}
	return m.dispatch(application.FocusPrompt{ID: live[0].ID})
}

func (m Model) focusLast() Model {
	live := m.state.Board.LivePrompts()
	if len(live) == 0 {
		return m
	}
	return m.dispatch(application.FocusPrompt{ID: live[len(live)-1].ID})
}

func (m Model) moveFocus(delta int) Model {
	live := m.state.Board.LivePrompts()
	if len(live) == 0 {
		return m
	}
	if m.state.FocusedID == nil {
		return m.dispatch(application.FocusPrompt{ID: live[0].ID})
	}
	idx := focusedIndex(live, m.state.FocusedID)
	idx += delta
	if idx < 0 {
		idx = 0
	}
	if idx > len(live)-1 {
		idx = len(live) - 1
	}
	return m.dispatch(application.FocusPrompt{ID: live[idx].ID})
}

// toggleMarked schaltet die Markierung des fokussierten Prompts um (Leertaste).
// Markierte Prompts überstimmen RemoveAfterSend, siehe Model.beginSend (internal/ui/model.go).
func (m Model) toggleMarked() Model {
	if m.state.FocusedID == nil {
		return m
	}
	return m.dispatch(application.ToggleMarked{ID: *m.state.FocusedID})
}

func (m Model) focusedContent() string {
	if m.state.FocusedID == nil {
		return ""
	}
	for _, t := range m.state.Board.LivePrompts() {
		if t.ID == *m.state.FocusedID {
			return t.Content
		}
	}
	return ""
}
