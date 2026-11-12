package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

type tab int

const (
	tabAll tab = iota
	tabGlobal
	tabLocal
	tabFixed
	tabLog
)

var tabLabels = []string{"Alle", "Global", "Lokal", "Fixiert", "Log"}

// tabCountWidth hält jede Reiterzelle unabhängig von der Anzahl gleich breit,
// damit die Reiterzeile beim Wechseln/Zählen nicht springt.
const tabCountWidth = 2

var (
	activeTabStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	inactiveTabStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	tabSeparator     = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│ ")
)

// tabFilter liefert den Sichtbarkeitsfilter eines Reiters für AppState.Visible.
// Log zeigt keine Prompts.
func tabFilter(t tab) func(domain.Prompt) bool {
	switch t {
	case tabAll:
		return func(domain.Prompt) bool { return true }
	case tabFixed:
		return func(p domain.Prompt) bool { return p.Marked }
	case tabGlobal:
		return func(p domain.Prompt) bool { return !p.IsLocal() }
	case tabLocal:
		return domain.Prompt.IsLocal
	default:
		return func(domain.Prompt) bool { return false }
	}
}

// applyTab merkt sich Fokus und Scroll-Fenster des alten Reiters und stellt die
// des neuen wieder her; ist der gemerkte Prompt dort nicht mehr sichtbar, greift
// der erste sichtbare Prompt.
func (m Model) applyTab(t tab) Model {
	m.tabFocus[m.tab] = m.state.FocusedID
	m.tabScroll[m.tab] = m.listScrollStart
	m.tab = t
	m.state.Visible = tabFilter(t)
	live := application.VisibleLive(m.state)
	if focusedIndex(live, m.tabFocus[t]) >= 0 {
		m.state.FocusedID = m.tabFocus[t]
		m.listScrollStart = m.tabScroll[t]
		return m
	}
	m.state.FocusedID = nil
	if len(live) > 0 {
		m.state.FocusedID = &live[0].ID
	}
	m.listScrollStart = 0
	return m
}

// switchTab wechselt um delta Reiter, am Rand geklemmt.
func (m Model) switchTab(delta int) Model {
	count := int(tabLog) + 1
	return m.applyTab(tab((int(m.tab) + delta + count) % count))
}

// newPromptAction liefert die CreatePrompt-Action für i/V im aktiven Reiter;
// false im Log-Reiter, der keine Prompts anlegt.
func (m Model) newPromptAction() (application.CreatePrompt, bool) {
	switch m.tab {
	case tabFixed:
		return application.CreatePrompt{Scope: domain.ScopeGlobal, Marked: true}, true
	case tabAll, tabGlobal:
		return application.CreatePrompt{Scope: domain.ScopeGlobal}, true
	case tabLocal:
		return application.CreatePrompt{Scope: domain.ScopeLocal}, true
	default:
		return application.CreatePrompt{}, false
	}
}

// tabBar rendert die Reiterzeile: Zellen fester Breite, durch │ getrennt.
func (m Model) tabBar() string {
	cells := make([]string, len(tabLabels))
	for i, label := range tabLabels {
		cells[i] = m.tabCell(tab(i), label)
	}
	return truncateToWidth(strings.Join(cells, tabSeparator), m.width)
}

// tabCell rendert einen Reiter als "Name 12 "; der aktive hebt sich nur per Farbe ab.
func (m Model) tabCell(t tab, label string) string {
	style := inactiveTabStyle
	if t == m.tab {
		style = activeTabStyle
	}
	return style.Render(fmt.Sprintf("%s %-*d", label, tabCountWidth, m.tabCount(t)))
}

// tabCount zählt die Prompts eines Reiters bzw. die Log-Einträge.
func (m Model) tabCount(t tab) int {
	if t == tabLog {
		return len(m.state.Board.Session.SentHistory)
	}
	visible := tabFilter(t)
	count := 0
	for _, p := range m.state.Board.LivePrompts() {
		if visible(p) {
			count++
		}
	}
	return count
}

// originPrefix markiert in den gemischten Reitern (Alle, Fixiert) die Herkunft eines Prompts.
func (m Model) originPrefix(p domain.Prompt) string {
	if m.tab != tabFixed && m.tab != tabAll {
		return ""
	}
	if p.IsLocal() {
		return "L "
	}
	return "G "
}

// emptyTabHint ist die Hinweiszeile für einen Reiter ohne sichtbare Prompts.
func (m Model) emptyTabHint() string {
	switch m.tab {
	case tabLocal:
		return "Keine lokalen Prompts — [i] legt qdrover.json an"
	case tabLog:
		return "Noch keine gesendeten Prompts"
	default:
		return "[i] neuer Prompt, [q] beenden, [h] Hilfe"
	}
}
