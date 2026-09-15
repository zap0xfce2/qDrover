package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"qdrover/internal/domain"
)

// sentTimestampFormat zeigt nur die Uhrzeit — reicht für eine TUI-Session an
// einem Tag und spart Footer-Platz gegenüber einem vollen Datum.
const sentTimestampFormat = "15:04:05"

// sentHistoryFadeColors: Farbrampe je Alters-Rang (0 = zuletzt gesendet).
// Alle Ränge nutzen feste xterm256-Codes statt Basis-ANSI (0-15): die sind
// terminal-theme-abhängig und unvorhersehbar. Rang 0 kräftiges Grün, Ränge
// 1-4 verblassen Richtung Grau mit gedämpftem Helligkeits-Boden (RGB
// 95,175,95 → 95,135,95 → 95,95,95 → 78,78,78) statt bis nahe Schwarz zu
// fallen — im Dark-Mode-Terminal per Preview verifiziert (User-Feedback,
// vorherige Rampen waren entweder am unteren Ende unlesbar oder nicht
// monoton unterscheidbar). Größe an domain.MaxSentHistoryEntries gebunden,
// da SentHistory nie mehr Einträge hält.
var sentHistoryFadeColors = []string{"34", "71", "65", "59", "239"}

// sentHistoryStyleForRank liefert den Style für einen Sendehistory-Eintrag
// nach Alters-Rang (0 = neuester). idx wird auf den letzten Farbeintrag
// geklemmt, falls MaxSentHistoryEntries künftig wächst, ohne dass
// sentHistoryFadeColors nachgezogen wird.
func sentHistoryStyleForRank(rank int) lipgloss.Style {
	idx := rank
	if idx >= len(sentHistoryFadeColors) {
		idx = len(sentHistoryFadeColors) - 1
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(sentHistoryFadeColors[idx]))
}

// prefixCommandStyle hebt den persistenten Plan-/Clear-Modus-Status im Footer hervor.
var prefixCommandStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))

// markedStyle hebt eine per Leertaste markierte Prompt-Zeile im Board hervor.
var markedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("6"))

// versionBannerStyle rendert die Startversion in Blau (fixer xterm256-Code,
// theme-unabhängig, statt des vormaligen Basis-ANSI-Grau "8").
var versionBannerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))

// withVersionBanner hängt "qDrover <version>" rechtsbündig als letzte
// Bildschirmzeile an, solange showVersionBanner() true liefert (bis zum
// ersten Tastendruck).
func (m Model) withVersionBanner(content string) string {
	if !m.showVersionBanner() {
		return content
	}
	return m.appendVersionBanner(content)
}

// withVersionBannerAlways hängt den Versions-Banner unabhängig von
// versionBannerDismissed an — für den Hilfe-Modus, der die Version immer
// zeigen soll.
func (m Model) withVersionBannerAlways(content string) string {
	if m.version == "" {
		return content
	}
	return m.appendVersionBanner(content)
}

// appendVersionBanner rendert und platziert den Banner (setzt m.version != ""
// voraus). Er braucht dafür keine eigene Zeile: er wird rechtsbündig an die
// bereits vorhandene letzte Inhaltszeile angehängt (lipgloss.Width statt
// len(), damit ANSI-Styling nicht mitgezählt wird), statt eine neue Zeile zu
// erzwingen — der Banner verbraucht damit kein Höhenbudget der Prompt-Liste.
// Ohne bekannte Breite (kein WindowSizeMsg empfangen, z. B. in Tests) wird
// wie bisher als eigene Zeile angehängt, da sich sonst keine sinnvolle Lücke
// berechnen lässt. Passt der Banner nicht mehr in die Restbreite der letzten
// Zeile, wird er weggelassen statt Overflow zu erzeugen.
func (m Model) appendVersionBanner(content string) string {
	label := "qDrover " + m.version
	if m.width <= 0 {
		return content + "\n" + versionBannerStyle.Render(label)
	}
	lines := strings.Split(content, "\n")
	last := len(lines) - 1
	gap := m.width - lipgloss.Width(lines[last]) - lipgloss.Width(label)
	if gap < 1 {
		return content
	}
	lines[last] += strings.Repeat(" ", gap) + versionBannerStyle.Render(label)
	return strings.Join(lines, "\n")
}

// prefixStatusLabel baut die Footer-Zeile für aktive Plan-/Clear-Modi:
// Anzeigereihenfolge Plan vor Clear (unabhängig von der Sendereihenfolge in
// activePrefixCommands, die Clear vor Plan sendet).
func (m Model) prefixStatusLabel() string {
	var labels []string
	if m.state.Board.Session.IsPlanModeActive() {
		labels = append(labels, "Plan-Modus: an")
	}
	if m.state.Board.Session.IsClearModeActive() {
		labels = append(labels, "Clear-Modus: an")
	}
	return strings.Join(labels, ", ")
}

// sentHistoryBlock rendert die persistierte Sendehistory (Session.SentHistory,
// max domain.MaxSentHistoryEntries Einträge) als Log mit Zeitstempel, neueste
// oben, älteste unten. Jeder Eintrag bleibt eine Zeile — zu lange Einträge
// werden wie in der Prompt-Liste per truncateToWidth gekürzt statt umgebrochen,
// man soll nur ungefähr sehen, was gesendet wurde. Jede Zeile wird einzeln
// nach Alters-Rang eingefärbt (sentHistoryStyleForRank) statt den ganzen
// Block einheitlich zu stylen — ältere Einträge verblassen sichtbar.
func (m Model) sentHistoryBlock() string {
	entries := m.state.Board.Session.SentHistory
	if len(entries) == 0 {
		return ""
	}
	lines := make([]string, 0, len(entries))
	for rank, i := 0, len(entries)-1; i >= 0; rank, i = rank+1, i-1 {
		e := entries[i]
		ts := time.UnixMilli(int64(e.SentAt)).Format(sentTimestampFormat)
		line := truncateToWidth(fmt.Sprintf("%s — %s", ts, e.Text), m.width)
		lines = append(lines, sentHistoryStyleForRank(rank).Render(line))
	}
	return strings.Join(lines, "\n")
}

// View liefert den Bildschirminhalt. Kürzt einen etwaigen trailing Newline
// aus renderView(): Bubble Tea zählt Zeilen per strings.Split("\n") und
// klippt bei zu vielen Zeilen von OBEN (siehe standard_renderer.go) — ein
// trailing "\n" (z. B. von fmt.Fprintln der letzten Prompt-Zeile) erzeugt
// dabei ein leeres Phantom-Element, das als zusätzliche Zeile zählt und bei
// knapper Terminalhöhe die oberste (eigentlich sichtbare) Zeile wegklippt.
func (m Model) View() string {
	return strings.TrimSuffix(m.renderView(), "\n")
}

func (m Model) renderView() string {
	if m.mode == modeHelp {
		return m.withVersionBannerAlways(helpText)
	}
	if m.mode == modeEdit {
		// Editor ist append-only, der Cursor steht also immer am Ende des Puffers.
		wrapped := wrapToWidth(m.editBuf, m.width)
		return fmt.Sprintf("Bearbeite Prompt:\n%s%s\n\n[Esc: speichern & verlassen]", wrapped, editCursor)
	}

	live := m.state.Board.LivePrompts()
	footer := strings.Join(m.footerLines(), "\n")
	if len(live) == 0 {
		hint := "[i] neuer Prompt, [q] beenden, [h] Hilfe"
		if footer != "" {
			hint += "\n" + footer
		}
		return m.withVersionBanner(hint)
	}

	start, end := m.visiblePromptRange(live)

	var b strings.Builder
	for _, t := range live[start:end] {
		cursor := "  "
		if m.state.FocusedID != nil && t.ID == *m.state.FocusedID {
			cursor = "> "
		}
		singleLine := strings.ReplaceAll(t.Content, "\n", " ")
		line := truncateToWidth(cursor+singleLine, m.width)
		if t.Marked {
			line = markedStyle.Render(line)
		}
		fmt.Fprintln(&b, line)
	}
	if footer != "" {
		fmt.Fprintf(&b, "%s\n", footer)
	}
	// TrimSuffix vor withVersionBanner (nicht erst im äußeren View()):
	// appendVersionBanner hängt den Banner an die letzte Zeile an — ohne
	// diesen Trim wäre das wegen des trailing "\n" ein leeres Phantom-Element
	// unterhalb der eigentlichen Status-/History-Zeile statt diese Zeile selbst.
	return m.withVersionBanner(strings.TrimSuffix(b.String(), "\n"))
}

// footerLines liefert die aktiven Footer-Abschnitte (Sendehistorie,
// Plan-/Clear-Status, Fehlermeldung) als einzelne Zeilenblöcke — leer, wenn
// nichts anzuzeigen ist. Einzige Quelle für sowohl das Rendering in View()
// als auch die Höhenmessung in footerHeight(), damit beide nie auseinanderlaufen.
// Plan-/Clear-Status steht bewusst nach der Historie (kurzer, fixer Text als
// letzte Footer-Zeile) — appendVersionBanner hängt den Versions-Banner an
// die letzte Zeile an, das lässt ihm zuverlässiger Platz als eine
// potenziell lange, näher an die Terminalbreite reichende History-Zeile.
func (m Model) footerLines() []string {
	var lines []string
	if block := m.sentHistoryBlock(); block != "" {
		lines = append(lines, block)
	}
	if label := m.prefixStatusLabel(); label != "" {
		lines = append(lines, prefixCommandStyle.Render(label))
	}
	if m.err != nil {
		lines = append(lines, fmt.Sprintf("Fehler: %s", m.err))
	}
	return lines
}

// footerHeight misst die tatsächliche Zeilenzahl, die footerLines() beim
// Rendern belegt, 0 wenn footerLines() leer ist. Keine eigene Trennzeile
// zur Prompt-Liste mehr (bewusst entfernt, damit die Zeile der Prompt-Liste
// zugutekommt statt ungenutzt zu bleiben).
func (m Model) footerHeight() int {
	footer := strings.Join(m.footerLines(), "\n")
	if footer == "" {
		return 0
	}
	return strings.Count(footer, "\n") + 1
}

// visiblePromptRange ermittelt die reservierte Höhe für Footer-Zeilen und
// liefert das verankerte Sichtfenster (siehe anchoredWindow). Rein lesend,
// mutiert m.listScrollStart nicht — die Fortschreibung übernimmt
// Model.syncScroll() zentral in Update().
func (m Model) visiblePromptRange(live []domain.Prompt) (int, int) {
	if m.height <= 0 {
		return 0, len(live)
	}
	maxVisible := m.height - m.footerHeight()
	return anchoredWindow(len(live), focusedIndex(live, m.state.FocusedID), maxVisible, m.listScrollStart)
}

// focusedIndex liefert den Index von focusedID in live, -1 falls nil oder
// nicht gefunden.
func focusedIndex(live []domain.Prompt, focusedID *domain.PromptID) int {
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

// editCursor markiert im Edit-Modus die (immer am Ende stehende) Cursor-Position.
const editCursor = "▋"

// truncateToWidth kürzt s auf width Runen und hängt "…" an, falls gekürzt wurde.
// width <= 0 bedeutet "kein WindowSizeMsg empfangen" — dann unverändert zurückgeben.
func truncateToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + "…"
}

// wrapToWidth bricht s bei width Runen pro Zeile um (Wortumbruch, keine
// Kürzung — anders als truncateToWidth, das für den Editor ungeeignet wäre).
// Bestehende Zeilenumbrüche (z. B. von Enter) bleiben als Absatzgrenzen
// erhalten. width <= 0 bedeutet "kein WindowSizeMsg empfangen" — dann
// unverändert zurückgeben.
func wrapToWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	// strings.Fields (in wrapParagraph) verschluckt folgende Leerzeichen —
	// die müssen wir separat sichern, sonst "verschwindet" ein per Leertaste
	// getippter Space am Ende des Editor-Puffers (genau da steht der Cursor).
	trimmed := strings.TrimRight(s, " ")
	trailingSpaces := s[len(trimmed):]
	paragraphs := strings.Split(trimmed, "\n")
	wrapped := make([]string, 0, len(paragraphs))
	for _, p := range paragraphs {
		wrapped = append(wrapped, wrapParagraph(p, width)...)
	}
	return strings.Join(wrapped, "\n") + trailingSpaces
}

// wrapParagraph wortumbricht einen einzelnen Absatz (ohne \n) auf width Runen.
func wrapParagraph(p string, width int) []string {
	words := strings.Fields(p)
	if len(words) == 0 {
		return []string{p}
	}

	var lines []string
	current := ""
	for _, word := range words {
		for _, chunk := range hardWrapWord(word, width) {
			if current == "" {
				current = chunk
				continue
			}
			if len([]rune(current))+1+len([]rune(chunk)) <= width {
				current += " " + chunk
			} else {
				lines = append(lines, current)
				current = chunk
			}
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// hardWrapWord bricht ein einzelnes Wort, das selbst länger als width ist,
// hart in width-Rune-Stücke (verhindert Endlos-Overflow bei z. B. langen
// URLs ohne Leerzeichen). Kürzere Wörter kommen unverändert als einziges
// Element zurück.
func hardWrapWord(word string, width int) []string {
	runes := []rune(word)
	if len(runes) <= width {
		return []string{word}
	}
	var chunks []string
	for len(runes) > width {
		chunks = append(chunks, string(runes[:width]))
		runes = runes[width:]
	}
	if len(runes) > 0 {
		chunks = append(chunks, string(runes))
	}
	return chunks
}

// anchoredWindow liefert den [start, end)-Ausschnitt eines vim-artig
// verankerten Scroll-Fensters: bleibt bei prevStart stehen, solange
// focusedIdx darin sichtbar ist, und springt nur um das nötige Minimum,
// wenn der Fokus den sichtbaren Rand verlässt — kein Re-Zentrieren bei
// jeder Fokusbewegung wie beim bisherigen visibleWindow. maxVisible<1 wird
// auf 1 geklemmt (nie ungeklippt alles zeigen, sonst sprengt die Liste bei
// sehr flachem Terminal die Bildschirmhöhe). Zeigt alles, wenn das Fenster
// eh alles fasst. prevStart wird zusätzlich an [0, total-maxVisible]
// geklemmt, falls die Liste seit dem letzten Sync geschrumpft ist.
func anchoredWindow(total, focusedIdx, maxVisible, prevStart int) (start, end int) {
	if maxVisible < 1 {
		maxVisible = 1
	}
	if total <= maxVisible {
		return 0, total
	}
	if focusedIdx < 0 {
		focusedIdx = 0
	}
	start = prevStart
	if start < 0 {
		start = 0
	}
	if start > total-maxVisible {
		start = total - maxVisible
	}
	if focusedIdx < start {
		start = focusedIdx
	}
	if focusedIdx >= start+maxVisible {
		start = focusedIdx - maxVisible + 1
	}
	return start, start + maxVisible
}
