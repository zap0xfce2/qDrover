package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

type tab int

const (
	tabFixed tab = iota
	tabGlobal
	tabLocal
	tabLog
)

var tabLabels = []string{"Fixiert", "Global", "Lokal", "Log"}

var activeTabStyle = lipgloss.NewStyle().Bold(true)

// tabFilter liefert den Sichtbarkeitsfilter eines Reiters für AppState.Visible.
// Log zeigt keine Prompts.
func tabFilter(t tab) func(domain.Prompt) bool {
	switch t {
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

// applyTab aktiviert t, fokussiert den ersten sichtbaren Prompt und setzt das
// Scroll-Fenster zurück.
func (m Model) applyTab(t tab) Model {
	m.tab = t
	m.state.Visible = tabFilter(t)
	m.state.FocusedID = nil
	if live := application.VisibleLive(m.state); len(live) > 0 {
		m.state.FocusedID = &live[0].ID
	}
	m.listScrollStart = 0
	return m
}

// switchTab wechselt um delta Reiter, am Rand geklemmt.
func (m Model) switchTab(delta int) Model {
	return m.applyTab(tab(min(max(int(m.tab)+delta, int(tabFixed)), int(tabLog))))
}

// newPromptAction liefert die CreatePrompt-Action für i/V im aktiven Reiter;
// false im Log-Reiter, der keine Prompts anlegt.
func (m Model) newPromptAction() (application.CreatePrompt, bool) {
	switch m.tab {
	case tabFixed:
		return application.CreatePrompt{Scope: domain.ScopeGlobal, Marked: true}, true
	case tabGlobal:
		return application.CreatePrompt{Scope: domain.ScopeGlobal}, true
	case tabLocal:
		return application.CreatePrompt{Scope: domain.ScopeLocal}, true
	default:
		return application.CreatePrompt{}, false
	}
}

// tabBar rendert die Reiterzeile, aktiver Reiter als [Name].
func (m Model) tabBar() string {
	labels := make([]string, len(tabLabels))
	for i, label := range tabLabels {
		labels[i] = label
		if tab(i) == m.tab {
			labels[i] = activeTabStyle.Render("[" + label + "]")
		}
	}
	return truncateToWidth(strings.Join(labels, "  "), m.width)
}

// originPrefix markiert im Fixiert-Reiter die Herkunft (global/lokal) eines Prompts.
func (m Model) originPrefix(p domain.Prompt) string {
	if m.tab != tabFixed {
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
