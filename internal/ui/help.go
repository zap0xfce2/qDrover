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
  V            Zwischenablage als neuen Prompt einfügen (braucht xclip/xsel/wl-clipboard + X11/Wayland)
  C            fokussierten Prompt in Zwischenablage kopieren (auch über SSH per OSC52)

Im Editor
  enter        speichern & verlassen
  esc          speichern & verlassen
  ctrl+j       neue Zeile einfügen

Senden an Herdr-Pane
  shift+↑/↓/←/→  senden, Prompt bleibt in der Liste
  s              nach oben senden, Prompt wird danach entfernt (Undo-fähig)
  S              /subtask + Text nach oben senden, Prompt wird danach entfernt (Undo-fähig)
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
