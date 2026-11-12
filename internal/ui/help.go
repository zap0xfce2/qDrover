package ui

import tea "github.com/charmbracelet/bubbletea"

// handleHelpKey verlässt die Hilfe-Seite bei jedem Tastendruck zurück in den Board-Modus.
func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.mode = modeBoard
	return m, nil
}

const helpText = `Tastenkürzel

Navigation
  j / ↓        einen Prompt nach unten (vom letzten zum ersten)
  k / ↑        einen Prompt nach oben (vom ersten zum letzten)
  ←/→          Reiter im Kreis wechseln (Alle · Global · Lokal · Fixiert · Log)

Bearbeiten
  i            neuen Prompt anlegen
  enter        fokussierten Prompt bearbeiten
  d            fokussierten Prompt löschen
  J            Prompt nach unten verschieben (nur Global/Lokal)
  K            Prompt nach oben verschieben (nur Global/Lokal)
  m            Prompt zwischen Global und Lokal hin- und herschieben
  V            Zwischenablage als neuen Prompt einfügen (braucht xclip/xsel/wl-clipboard + X11/Wayland)
  C            fokussierten Prompt in Zwischenablage kopieren (auch über SSH per OSC52)

Im Editor
  enter        speichern & verlassen
  esc          speichern & verlassen
  ctrl+j       neue Zeile einfügen
  ←/→/↑/↓      Cursor bewegen
  home/end     Zeilenanfang/-ende (auch ctrl+a/ctrl+e)
  alt+←/→      wortweise springen

Senden an Herdr-Pane
  shift+↑/↓/←/→  senden, Prompt bleibt in der Liste
  s              nach oben senden, Prompt wird danach entfernt (Undo-fähig)
  leertaste      Prompt fixieren/lösen (cyan, erscheint im Reiter Fixiert) — überstimmt Löschen bei s
  p              Plan-Modus umschalten (Ziel bekommt vor jedem Send erst /plan), standardmäßig an
  c              Clear-Modus umschalten (Ziel bekommt vor jedem Send erst /clear), standardmäßig an

Verlauf
  u            Undo
  R            Redo
  r            Dateien neu einlesen (z. B. nach Austausch von qdrover.json)

Sonstiges
  h            diese Hilfe anzeigen
  q / ctrl+c   beenden

[eine beliebige Taste schließt die Hilfe]`
