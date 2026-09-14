package ui

import tea "github.com/charmbracelet/bubbletea"

// handleHelpKey verlässt die Hilfe-Seite bei jedem Tastendruck zurück in den Board-Modus.
func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	return m, nil
}

const helpText = `Tastenkürzel

Navigation
  j / ↓        einen Prompt nach unten
  k / ↑        einen Prompt nach oben
  g            zum ersten Prompt
  G            zum letzten Prompt

Bearbeiten
  i            neuen Prompt anlegen
  enter        fokussierten Prompt bearbeiten
  d            fokussierten Prompt löschen
  J            Prompt nach unten verschieben
  K            Prompt nach oben verschieben

Im Editor
  esc          speichern & verlassen
  enter        neue Zeile einfügen

Senden an Herdr-Pane
  shift+↑/↓/←/→  senden, Prompt bleibt in der Liste
  s              nach oben senden, Prompt wird danach entfernt (Undo-fähig)
  S              nach oben senden, Prompt bleibt in der Liste
  leertaste      Prompt markieren/entmarkieren (cyan) — überstimmt Löschen bei s
  p              Plan-Modus umschalten (Ziel bekommt vor jedem Send erst /plan), standardmäßig an
  c              Clear-Modus umschalten (Ziel bekommt vor jedem Send erst /clear), standardmäßig an

Verlauf
  u            Undo
  r            Redo

Sonstiges
  h            diese Hilfe anzeigen
  q / ctrl+c   beenden

[eine beliebige Taste schließt die Hilfe]`
