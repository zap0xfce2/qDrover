package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/ports"
)

// logTabKeys sind die einzigen Tasten, die der Log-Reiter annimmt; alles
// andere würde Prompts bearbeiten, senden oder die Zwischenablage berühren.
var logTabKeys = map[string]bool{
	"left": true, "right": true, "h": true, "q": true, "ctrl+c": true,
	"p": true, "c": true,
}

func (m Model) handleBoardKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.tab == tabLog && !logTabKeys[msg.String()] {
		return m, nil
	}
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
	case "left":
		return m.switchTab(-1), nil
	case "right":
		return m.switchTab(1), nil
	case "down", "j":
		return m.moveFocus(1), nil
	case "up", "k":
		return m.moveFocus(-1), nil
	case "enter":
		return m.startEdit(m.focusedContent(), false), nil
	case "i":
		return m.createPrompt(""), nil
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
		return m.pasteAsPrompt(), nil
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

// movePrompt tauscht den fokussierten Prompt mit seinem sichtbaren Nachbarn;
// nur in Global/Lokal, da Fixiert beide Scopes mischt und Log keine Prompts hat.
func (m Model) movePrompt(delta int) Model {
	if m.state.FocusedID == nil || (m.tab != tabGlobal && m.tab != tabLocal) {
		return m
	}
	live := application.VisibleLive(m.state)
	target := focusedIndex(live, m.state.FocusedID) + delta
	if target < 0 || target >= len(live) {
		return m
	}
	return m.dispatch(application.MovePrompt{ID: *m.state.FocusedID, To: live[target].Position})
}

// createPrompt legt im aktiven Reiter einen Prompt an und öffnet ihn im
// Editor; No-op im Log-Reiter.
func (m Model) createPrompt(content string) Model {
	action, ok := m.newPromptAction()
	if !ok {
		return m
	}
	return m.dispatch(action).startEdit(content, true)
}

// pasteAsPrompt legt die Zwischenablage als neuen Prompt an (Taste V).
func (m Model) pasteAsPrompt() Model {
	if _, ok := m.newPromptAction(); !ok {
		return m
	}
	content, err := m.clipboard.ReadAll()
	if err != nil {
		m.err = err
		return m
	}
	return m.createPrompt(content)
}

func (m Model) focusFirst() Model {
	live := application.VisibleLive(m.state)
	if len(live) == 0 {
		return m
	}
	return m.dispatch(application.FocusPrompt{ID: live[0].ID})
}

func (m Model) focusLast() Model {
	live := application.VisibleLive(m.state)
	if len(live) == 0 {
		return m
	}
	return m.dispatch(application.FocusPrompt{ID: live[len(live)-1].ID})
}

func (m Model) moveFocus(delta int) Model {
	live := application.VisibleLive(m.state)
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
	for _, t := range application.VisibleLive(m.state) {
		if t.ID == *m.state.FocusedID {
			return t.Content
		}
	}
	return ""
}
