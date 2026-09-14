package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/rivo/uniseg"

	"qdrover/internal/application"
)

func trimLastGrapheme(s string) string {
	if s == "" {
		return s
	}
	var lastBoundary int
	state := -1
	rest := s
	pos := 0
	for len(rest) > 0 {
		var cluster string
		cluster, rest, _, state = uniseg.StepString(rest, state)
		lastBoundary = pos
		pos += len(cluster)
	}
	return s[:lastBoundary]
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = modeBoard
		if m.state.FocusedID != nil {
			if m.editingNewPrompt && strings.TrimSpace(m.editBuf) == "" {
				m = m.dispatch(application.DeletePrompt{ID: *m.state.FocusedID})
			} else {
				m = m.dispatch(application.EditPrompt{ID: *m.state.FocusedID, Content: m.editBuf})
			}
		}
		m.editingNewPrompt = false
		return m, nil
	case tea.KeyEnter:
		m.editBuf += "\n"
		return m, nil
	case tea.KeyBackspace:
		m.editBuf = trimLastGrapheme(m.editBuf)
		return m, nil
	case tea.KeyRunes:
		m.editBuf += string(msg.Runes)
		return m, nil
	case tea.KeySpace:
		m.editBuf += " "
		return m, nil
	}
	return m, nil
}
