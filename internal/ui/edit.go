package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"qdrover/internal/application"
)

// editChromeLines: Überschrift, Leerzeile und Hinweiszeile rund um das Editfeld.
const editChromeLines = 3

// defaultEditorHeight gilt, solange noch kein WindowSizeMsg kam (z. B. in Tests).
const defaultEditorHeight = 10

// newEditor baut ein schlichtes, unbegrenztes Editfeld ohne Rahmen/Zeilennummern.
// Enter ist bewusst nicht "neue Zeile" (speichert, siehe handleEditKey), das übernimmt ctrl+j.
func newEditor() textarea.Model {
	ta := textarea.New()
	ta.Prompt = ""
	ta.Placeholder = ""
	ta.ShowLineNumbers = false
	ta.CharLimit = 0
	ta.MaxHeight = 0
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.KeyMap.InsertNewline = key.NewBinding(key.WithKeys("ctrl+j"))
	// Statischer Cursor: Blinken bräuchte einen laufenden Tick-Cmd, ohne bleibt er sonst hängen.
	ta.Cursor.SetMode(cursor.CursorStatic)
	ta.Focus()
	return ta
}

// startEdit wechselt in den Edit-Modus mit content vorbefüllt, Cursor am Ende.
func (m Model) startEdit(content string, isNewPrompt bool) Model {
	m.mode = modeEdit
	m.editingNewPrompt = isNewPrompt
	m.editor = newEditor()
	m.editor.SetValue(content)
	return m.resizeEditor()
}

func (m Model) resizeEditor() Model {
	height := defaultEditorHeight
	if m.height > 0 {
		height = max(m.height-editChromeLines, 1)
	}
	m.editor.SetHeight(height)
	if m.width > 0 {
		m.editor.SetWidth(m.width)
	}
	return m
}

func (m Model) handleEditKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEsc || msg.Type == tea.KeyEnter {
		return m.commitEdit(), nil
	}
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

// commitEdit speichert den Editor-Inhalt und verlässt den Edit-Modus (löscht
// stattdessen bei leerem, neu angelegtem Prompt) — genutzt von Enter und Esc.
func (m Model) commitEdit() Model {
	m.mode = modeBoard
	content := m.editor.Value()
	if m.state.FocusedID != nil {
		if m.editingNewPrompt && strings.TrimSpace(content) == "" {
			m = m.dispatch(application.DeletePrompt{ID: *m.state.FocusedID})
		} else {
			m = m.dispatch(application.EditPrompt{ID: *m.state.FocusedID, Content: content})
		}
	}
	m.editingNewPrompt = false
	return m
}
