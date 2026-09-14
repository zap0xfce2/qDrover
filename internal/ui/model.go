package ui

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

// herdrCallTimeout deckt auch kombinierten Plan- und Clear-Modus ab (bis zu
// drei Herdr-Aufrufe + zwei prefixSettleDelay statt nur einem Aufruf).
const herdrCallTimeout = 8 * time.Second

// prefixCommandPlan und prefixCommandClear sind die Präfix-Texte für die
// unabhängig per Taste p/c umschaltbaren, in domain.Session persistierten
// Modi PlanModeActive/ClearModeActive.
const (
	prefixCommandPlan  = "/plan"
	prefixCommandClear = "/clear"
)

type mode int

const (
	modeBoard mode = iota
	modeEdit
	modeHelp
)

type Model struct {
	state    application.AppState
	executor *application.Executor
	clock    ports.Clock
	ids      ports.IDGenerator
	mode     mode
	editBuf  string
	// editingNewPrompt: true während der Edit-Session eines frisch mit "i"
	// angelegten Prompts — steuert, ob ein beim Verlassen des Edit-Modus
	// noch leerer Prompt automatisch gelöscht wird (siehe handleEditKey).
	editingNewPrompt bool
	width            int
	height           int
	err              error
	version          string
	// versionBannerDismissed: der Start-Banner unten rechts bleibt sichtbar,
	// bis irgendeine Taste gedrückt wird (siehe Update, showVersionBanner).
	versionBannerDismissed bool
	// listScrollStart: persistenter Start-Index des sichtbaren
	// Prompt-Fensters für vim-artiges verankertes statt zentriertes
	// Scrollen (siehe anchoredWindow in view.go, syncScroll unten).
	listScrollStart int
}

func New(state application.AppState, executor *application.Executor, clock ports.Clock, ids ports.IDGenerator) Model {
	return Model{state: state, executor: executor, clock: clock, ids: ids, mode: modeBoard}
}

// WithVersion setzt die per ldflags gebaute Versionsnummer für den
// Start-Banner unten rechts (siehe showVersionBanner).
func (m Model) WithVersion(version string) Model {
	m.version = version
	return m
}

// showVersionBanner meldet, ob der Versions-Banner im normalen Board-/Edit-Pfad
// noch sichtbar sein soll: nur mit gesetzter Version und solange noch keine
// Taste gedrückt wurde. Der Hilfe-Modus zeigt die Version unabhängig davon
// immer (siehe view.go withVersionBannerAlways).
func (m Model) showVersionBanner() bool {
	return m.version != "" && !m.versionBannerDismissed
}

// activePrefixCommands liefert die aktiven Präfix-Kommandos in Sendereihenfolge:
// erst /clear, dann /plan.
func (m Model) activePrefixCommands() []string {
	var cmds []string
	if m.state.Board.Session.IsClearModeActive() {
		cmds = append(cmds, prefixCommandClear)
	}
	if m.state.Board.Session.IsPlanModeActive() {
		cmds = append(cmds, prefixCommandPlan)
	}
	return cmds
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) dispatch(action application.Action) Model {
	newState, effects, err := application.Reduce(m.state, action, m.clock, m.ids)
	if err != nil {
		m.err = err
		return m
	}
	m.state = newState
	m.err = nil
	ctx, cancel := context.WithTimeout(context.Background(), herdrCallTimeout)
	defer cancel()
	if execErr := m.executor.Execute(ctx, effects); execErr != nil {
		m.err = execErr
		return m
	}
	m = m.recordSentHistory(effects)
	return m.removeSentPromptsIfRequested(action, effects)
}

// recordSentHistory dispatcht für jeden erfolgreich ausgeführten
// SendDispatch-Effect ein RecordSentText, das den gesendeten Text (Zeilenumbrüche
// durch Leerzeichen ersetzt) an die persistierte Session.SentHistory anhängt
// (Footer-Anzeige in View()).
func (m Model) recordSentHistory(effects []application.Effect) Model {
	for _, eff := range effects {
		dispatchEffect, ok := eff.(application.SendDispatch)
		if !ok {
			continue
		}
		singleLine := strings.ReplaceAll(dispatchEffect.Text, "\n", " ")
		m = m.dispatch(application.RecordSentText{Text: singleLine})
	}
	return m
}

// removeSentPromptsIfRequested löscht nach einem erfolgreichen Senden mit
// RemoveAfterSend die gesendeten Prompts (Soft-Delete, per Undo
// wiederherstellbar) — ein DeletePrompt-Dispatch pro Prompt, damit Undo
// wie gewohnt funktioniert. Markierte Prompts (Model.toggleMarked) überstimmen
// das: sie bleiben trotz RemoveAfterSend erhalten, bis die Markierung erneut
// per Leertaste entfernt wird.
func (m Model) removeSentPromptsIfRequested(action application.Action, effects []application.Effect) Model {
	send, ok := action.(application.SendSelectionToPane)
	if !ok || !send.RemoveAfterSend {
		return m
	}
	for _, eff := range effects {
		dispatchEffect, ok := eff.(application.SendDispatch)
		if !ok {
			continue
		}
		for _, id := range dispatchEffect.PromptIDs {
			if m.isMarked(id) {
				continue
			}
			m = m.dispatch(application.DeletePrompt{ID: id})
		}
	}
	return m
}

// isMarked prüft den aktuellen Markierungs-Status eines lebenden Prompts.
func (m Model) isMarked(id domain.PromptID) bool {
	for _, t := range m.state.Board.LivePrompts() {
		if t.ID == id {
			return t.Marked
		}
	}
	return false
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m.syncScroll(), nil
	case tea.KeyMsg:
		m.versionBannerDismissed = true
		next, cmd := m.handleKey(msg)
		if nm, ok := next.(Model); ok {
			next = nm.syncScroll()
		}
		return next, cmd
	}
	return m, nil
}

// syncScroll aktualisiert m.listScrollStart anhand von aktuellem Fokus und
// Terminalhöhe (vim-artiges verankertes statt zentriertes Scrollen, siehe
// anchoredWindow in view.go). Muss nach jeder Fokus-/Listenänderung und nach
// jedem Resize laufen — reine Zustands-Fortschreibung, kein Rendering.
// No-op außerhalb des Board-Modus oder ohne bekannte Höhe.
func (m Model) syncScroll() Model {
	if m.mode != modeBoard || m.height <= 0 {
		return m
	}
	live := m.state.Board.LivePrompts()
	maxVisible := m.height - m.footerHeight()
	m.listScrollStart, _ = anchoredWindow(len(live), focusedIndex(live, m.state.FocusedID), maxVisible, m.listScrollStart)
	return m
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case modeEdit:
		return m.handleEditKey(msg)
	case modeHelp:
		return m.handleHelpKey(msg)
	default:
		return m.handleBoardKey(msg)
	}
}
