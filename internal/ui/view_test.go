package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

func TestView_BoardMode_ListsPromptContents(t *testing.T) {
	m := newTestModel()
	out := m.View()

	if !strings.Contains(out, "erste Idee") || !strings.Contains(out, "zweite Idee") {
		t.Fatalf("erwarte beide Prompt-Inhalte in View-Output, habe:\n%s", out)
	}
}

func TestView_BoardMode_MultilinePrompt_ShowsFullContentAsOneLine(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee\nzweite Zeile", 100)
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{})

	out := m.View()

	if strings.Contains(out, "erste Idee\nzweite Zeile") {
		t.Fatalf("erwarte Content ohne Zeilenumbruch in View-Output, habe:\n%s", out)
	}
	if !strings.Contains(out, "erste Idee zweite Zeile") {
		t.Fatalf("erwarte vollständigen Content als eine Zeile in View-Output, habe:\n%s", out)
	}
}

func TestView_BoardMode_MarkedPrompt_IsStyledWithMarkedStyle(t *testing.T) {
	m := newTestModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown}) // Fokus auf t1
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace}) // t1 markieren
	m = updated.(Model)

	out := m.View()
	expectedLine := markedStyle.Render("> erste Idee")
	if !strings.Contains(out, expectedLine) {
		t.Fatalf("erwarte mit markedStyle gerenderte Zeile für markierten Prompt, habe:\n%s", out)
	}
}

func TestView_EditMode_ShowsEditBuffer(t *testing.T) {
	m := newTestModel()
	m.mode = modeEdit
	m.editBuf = "in Bearbeitung"

	out := m.View()
	if !strings.Contains(out, "in Bearbeitung") {
		t.Fatalf("erwarte Edit-Buffer in View-Output, habe:\n%s", out)
	}
}

func TestView_EditMode_ShowsCursorRightAfterBuffer(t *testing.T) {
	m := newTestModel()
	m.mode = modeEdit
	m.editBuf = "in Bearbeitung"

	out := m.View()
	if !strings.Contains(out, "in Bearbeitung"+editCursor) {
		t.Fatalf("erwarte Cursor direkt nach dem Edit-Buffer, habe:\n%s", out)
	}
}

func TestView_BoardMode_PlanModeActive_ShowsStatusInFooter(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if !strings.Contains(out, "Plan-Modus: an") {
		t.Fatalf("erwarte 'Plan-Modus: an' im Footer, habe:\n%s", out)
	}
}

func TestView_BoardMode_ClearModeActive_ShowsStatusInFooter(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if !strings.Contains(out, "Clear-Modus: an") {
		t.Fatalf("erwarte 'Clear-Modus: an' im Footer, habe:\n%s", out)
	}
}

func TestView_BoardMode_BothModesActive_ShowsCombinedStatusInFooter(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if !strings.Contains(out, "Plan-Modus: an, Clear-Modus: an") {
		t.Fatalf("erwarte kombinierte Statuszeile im Footer, habe:\n%s", out)
	}
}

func TestView_BoardMode_NoPrefixActive_ShowsNoFooterLine(t *testing.T) {
	m := newTestModel()
	planOff, clearOff := false, false
	m.state.Board.Session.PlanModeActive = &planOff
	m.state.Board.Session.ClearModeActive = &clearOff

	out := m.View()
	if strings.Contains(out, "-Modus:") {
		t.Fatalf("erwarte keine Modus-Zeile ohne aktives Präfix-Kommando, habe:\n%s", out)
	}
}

func TestView_BoardMode_ShowsSentHistoryInFooter(t *testing.T) {
	m := newTestModel()
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "erste Idee"})

	out := m.View()
	wantTimestamp := time.UnixMilli(0).Format(sentTimestampFormat)
	if !strings.Contains(out, wantTimestamp+" — erste Idee") {
		t.Fatalf("erwarte Sendehistory-Zeile mit Zeitstempel im Footer, habe:\n%s", out)
	}
}

func TestView_BoardMode_NoSentHistory_ShowsNoFooterLine(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if strings.Contains(out, "—") {
		t.Fatalf("erwarte keine Sendehistory-Zeile ohne Einträge, habe:\n%s", out)
	}
}

func TestView_BoardMode_ShowsMultipleSentHistoryEntriesInOrder(t *testing.T) {
	m := newTestModel()
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "erste Idee"})
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "zweite Idee"})

	out := m.View()
	firstIdx := strings.Index(out, "— erste Idee")
	secondIdx := strings.Index(out, "— zweite Idee")
	if firstIdx == -1 || secondIdx == -1 || firstIdx < secondIdx {
		t.Fatalf("erwarte beide Historyeinträge in Reihenfolge (neueste zuerst), habe:\n%s", out)
	}
}

func TestView_BoardMode_OlderSentHistoryEntry_IsStyledWithFadedColor(t *testing.T) {
	m := newTestModel()
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "erste Idee"})
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "zweite Idee"})

	out := m.View()
	wantTimestamp := time.UnixMilli(0).Format(sentTimestampFormat)
	newest := sentHistoryStyleForRank(0).Render(wantTimestamp + " — zweite Idee")
	older := sentHistoryStyleForRank(1).Render(wantTimestamp + " — erste Idee")
	if !strings.Contains(out, newest) {
		t.Fatalf("erwarte neuesten Eintrag in Rang-0-Farbe, habe:\n%s", out)
	}
	if !strings.Contains(out, older) {
		t.Fatalf("erwarte älteren Eintrag in abgeschwächter Farbe, habe:\n%s", out)
	}
}

func TestView_EmptyBoard_ShowsSentHistoryInFooter(t *testing.T) {
	m := newTestModel()
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "erste Idee"})
	m.state.Board.Prompts = nil

	out := m.View()
	wantTimestamp := time.UnixMilli(0).Format(sentTimestampFormat)
	if !strings.Contains(out, wantTimestamp+" — erste Idee") {
		t.Fatalf("erwarte Sendehistory-Zeile im Footer auch bei leerem Board, habe:\n%s", out)
	}
}

func TestView_BoardMode_LongSentHistoryEntry_TruncatesToOneLine(t *testing.T) {
	m := newTestModel()
	m.width = 20
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "ein sehr langer gesendeter Text"})

	out := m.View()
	if !strings.Contains(out, "…") {
		t.Fatalf("erwarte gekürzten Sendehistory-Eintrag mit Ellipse, habe:\n%s", out)
	}
	if strings.Contains(out, "ein sehr langer gesendeter Text") {
		t.Fatalf("erwarte gekürzten statt vollständigen Text im Footer, habe:\n%s", out)
	}
}

func TestView_HelpMode_ShowsShortcuts(t *testing.T) {
	m := newTestModel()
	m.mode = modeHelp

	out := m.View()
	if !strings.Contains(out, "Tastenkürzel") || !strings.Contains(out, "Navigation") {
		t.Fatalf("erwarte Tastenkürzel-Übersicht in View-Output, habe:\n%s", out)
	}
}

func TestView_EmptyBoard_ShowsHint(t *testing.T) {
	m := New(
		emptyState(),
		m0Executor(),
		fakeClock{now: 100},
		&fakeIDGen{},
	)
	out := m.View()
	if !strings.Contains(out, "[i] neuer Prompt") {
		t.Fatalf("erwarte Hinweis auf leeres Board, habe:\n%s", out)
	}
}

func TestView_BoardMode_MultipleFooterSections_NoDoubleBlankLines(t *testing.T) {
	m := newTestModel()
	m.state.Board.Session = m.state.Board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 0, Text: "erste Idee"})
	m.err = fmt.Errorf("Testfehler")

	out := m.View()
	if strings.Contains(out, "\n\n\n") {
		t.Fatalf("erwarte keine doppelten Leerzeilen zwischen Footer-Sektionen, habe:\n%q", out)
	}
}

func TestView_BoardMode_OutputNeverExceedsTerminalHeight(t *testing.T) {
	board := domain.Board{}
	for i := 0; i < 20; i++ {
		board = board.AddPrompt(domain.PromptID(fmt.Sprintf("t%d", i)), fmt.Sprintf("Prompt %d", i), 100)
	}
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)
	// Tastendruck: Banner weg (füllt sonst die Phantom-Zeile, siehe View()-Kommentar).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	out := m.View()
	lineCount := strings.Count(out, "\n") + 1
	if lineCount > m.height {
		t.Fatalf("erwarte höchstens %d Zeilen (sonst clippt Bubble Tea von oben), habe %d:\n%q", m.height, lineCount, out)
	}
}

func TestView_EmptyBoard_ShowsActivePrefixLabel(t *testing.T) {
	m := New(
		emptyState(),
		m0Executor(),
		fakeClock{now: 100},
		&fakeIDGen{},
	)

	out := m.View()
	if !strings.Contains(out, "Plan-Modus: an") {
		t.Fatalf("erwarte Plan-Modus-Label auch bei leerem Board, habe:\n%s", out)
	}
}

func TestTruncateToWidth_LongerThanWidth_TruncatesWithEllipsis(t *testing.T) {
	out := truncateToWidth("abcdefghij", 5)
	if out != "abcd…" {
		t.Fatalf("erwarte \"abcd…\", habe %q", out)
	}
	if len([]rune(out)) != 5 {
		t.Fatalf("erwarte Rune-Länge 5, habe %d", len([]rune(out)))
	}
}

func TestTruncateToWidth_ShorterThanWidth_ReturnsUnchanged(t *testing.T) {
	out := truncateToWidth("abc", 10)
	if out != "abc" {
		t.Fatalf("erwarte unverändert \"abc\", habe %q", out)
	}
}

func TestTruncateToWidth_ZeroOrNegativeWidth_ReturnsUnchanged(t *testing.T) {
	if out := truncateToWidth("abcdef", 0); out != "abcdef" {
		t.Fatalf("erwarte unverändert bei width=0, habe %q", out)
	}
	if out := truncateToWidth("abcdef", -1); out != "abcdef" {
		t.Fatalf("erwarte unverändert bei width<0, habe %q", out)
	}
}

func TestWrapToWidth_ShortText_ReturnsUnchanged(t *testing.T) {
	out := wrapToWidth("kurzer text", 20)
	if out != "kurzer text" {
		t.Fatalf("erwarte unverändert, habe %q", out)
	}
}

func TestWrapToWidth_LongText_BreaksAtWordBoundaries(t *testing.T) {
	out := wrapToWidth("eins zwei drei vier", 9)
	want := "eins zwei\ndrei vier"
	if out != want {
		t.Fatalf("erwarte %q, habe %q", want, out)
	}
}

func TestWrapToWidth_SingleWordLongerThanWidth_HardWraps(t *testing.T) {
	out := wrapToWidth("abcdefghij", 4)
	want := "abcd\nefgh\nij"
	if out != want {
		t.Fatalf("erwarte %q, habe %q", want, out)
	}
}

func TestWrapToWidth_ExistingNewlines_StayAsParagraphBreaks(t *testing.T) {
	out := wrapToWidth("eins zwei\ndrei", 9)
	want := "eins zwei\ndrei"
	if out != want {
		t.Fatalf("erwarte %q, habe %q", want, out)
	}
}

func TestWrapToWidth_ZeroOrNegativeWidth_ReturnsUnchanged(t *testing.T) {
	if out := wrapToWidth("eins zwei drei", 0); out != "eins zwei drei" {
		t.Fatalf("erwarte unverändert bei width=0, habe %q", out)
	}
	if out := wrapToWidth("eins zwei drei", -1); out != "eins zwei drei" {
		t.Fatalf("erwarte unverändert bei width<0, habe %q", out)
	}
}

func TestWrapToWidth_PreservesSingleTrailingSpace(t *testing.T) {
	out := wrapToWidth("hallo ", 20)
	if out != "hallo " {
		t.Fatalf("erwarte 'hallo ' (mit Trailing-Space), habe %q", out)
	}
}

func TestWrapToWidth_PreservesMultipleTrailingSpaces(t *testing.T) {
	out := wrapToWidth("hallo   ", 20)
	if out != "hallo   " {
		t.Fatalf("erwarte 'hallo   ' (3 Trailing-Spaces), habe %q", out)
	}
}

func TestView_EditMode_CursorMovesAfterSpaceKey(t *testing.T) {
	m := newTestModel()
	m.mode = modeEdit
	m.width = 40
	m.editBuf = "hallo"

	outBefore := m.View()
	if !strings.Contains(outBefore, "hallo"+editCursor) {
		t.Fatalf("erwarte Cursor direkt hinter 'hallo', habe:\n%s", outBefore)
	}

	m.editBuf = "hallo " // Leertaste gedrückt

	outAfter := m.View()
	if !strings.Contains(outAfter, "hallo "+editCursor) {
		t.Fatalf("erwarte Cursor nach dem Leerzeichen, nicht direkt hinter 'hallo' — Bug: %q enthält %q statt 'hallo %s'", outAfter, "hallo"+editCursor, editCursor)
	}
}

func TestView_EditMode_WrapsLongBufferAtWindowWidth(t *testing.T) {
	m := newTestModel()
	m.mode = modeEdit
	m.width = 9
	m.editBuf = "eins zwei drei vier"

	out := m.View()
	if !strings.Contains(out, "eins zwei\ndrei vier") {
		t.Fatalf("erwarte umgebrochenen Buffer in View-Output, habe:\n%s", out)
	}
}

func TestView_VersionBanner_ShownBeforeAnyKeyPress(t *testing.T) {
	m := newTestModel().WithVersion("v2609121234")

	out := m.View()
	if !strings.Contains(out, "qDrover v2609121234") {
		t.Fatalf("erwarte Versions-Banner vor dem ersten Tastendruck, habe:\n%s", out)
	}
}

func TestView_VersionBanner_HiddenAfterKeyPress(t *testing.T) {
	m := newTestModel().WithVersion("v2609121234")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)

	out := m.View()
	if strings.Contains(out, "qDrover") {
		t.Fatalf("erwarte kein Versions-Banner nach einem Tastendruck, habe:\n%s", out)
	}
}

func TestView_VersionBanner_HiddenWithoutVersion(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if strings.Contains(out, "qDrover") {
		t.Fatalf("erwarte kein Versions-Banner ohne gesetzte Version, habe:\n%s", out)
	}
}

func TestAppendVersionBanner_WithKnownWidth_InlinesIntoLastLine(t *testing.T) {
	m := newTestModel().WithVersion("v1")
	m.width = 60 // Statuszeile ("Plan-Modus: an, Clear-Modus: an") + Banner müssen nebeneinander passen

	out := m.View()
	lines := strings.Split(out, "\n")
	last := lines[len(lines)-1]
	if !strings.Contains(last, "qDrover v1") {
		t.Fatalf("erwarte Banner in der letzten Zeile statt einer eigenen, habe:\n%q", out)
	}
	if strings.Count(out, "qDrover") != 1 {
		t.Fatalf("erwarte Banner genau einmal, habe:\n%q", out)
	}
	// newTestModel() hat Plan-Modus aktiv (Footer nicht leer) — der Banner
	// muss wirklich auf derselben Zeile wie die Statuszeile landen, nicht auf
	// einer eigenen (leeren) Zeile darunter.
	if !strings.Contains(last, "Plan-Modus: an") {
		t.Fatalf("erwarte Banner auf derselben Zeile wie 'Plan-Modus: an' statt auf eigener Zeile, habe:\n%q", out)
	}
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	withoutBanner := updated.(Model).View()
	if strings.Count(out, "\n") != strings.Count(withoutBanner, "\n") {
		t.Fatalf("erwarte gleiche Zeilenzahl mit/ohne Banner (Banner braucht keine eigene Zeile), habe %d vs %d Newlines", strings.Count(out, "\n"), strings.Count(withoutBanner, "\n"))
	}
}

func TestView_BoardMode_NoBlankLineBetweenPromptsAndFooter(t *testing.T) {
	m := newTestModel()

	out := m.View()
	if strings.Contains(out, "\n\n") {
		t.Fatalf("erwarte keine Leerzeile zwischen Prompt-Liste und Footer, habe:\n%q", out)
	}
}

func TestAppendVersionBanner_NoRoom_OmitsInsteadOfOverflow(t *testing.T) {
	m := newTestModel().WithVersion("v1")
	m.width = 5

	withBanner := m.View()
	m.versionBannerDismissed = true
	withoutBanner := m.View()

	if strings.Contains(withBanner, "qDrover") {
		t.Fatalf("erwarte weggelassenen Banner bei zu wenig Platz statt Overflow, habe:\n%q", withBanner)
	}
	if withBanner != withoutBanner {
		t.Fatalf("erwarte identischen Output mit/ohne Banner bei zu wenig Platz, habe:\n%q\nvs\n%q", withBanner, withoutBanner)
	}
}

func TestView_HelpMode_ShowsVersionEvenAfterKeyPress(t *testing.T) {
	m := newTestModel().WithVersion("v2609121234")
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	m = updated.(Model)
	m.mode = modeHelp

	out := m.View()
	if !strings.Contains(out, "qDrover v2609121234") {
		t.Fatalf("erwarte Versions-Banner im Hilfe-Modus auch nach Tastendruck, habe:\n%s", out)
	}
}

func TestAnchoredWindow(t *testing.T) {
	cases := []struct {
		name                              string
		total, focused, maxVis, prevStart int
		wantStart, wantEnd                int
	}{
		{"passt komplett", 3, 1, 5, 0, 0, 3},
		{"maxVisible<1 klemmt auf 1 statt alles zu zeigen", 5, 2, 0, 0, 2, 3},
		{"Fokus bleibt im Fenster: start unveraendert (kein Re-Zentrieren)", 10, 4, 4, 2, 2, 6},
		{"Fokus verlaesst unten: minimaler Sprung", 10, 6, 4, 2, 3, 7},
		{"Fokus verlaesst oben (z.B. nach g): Sprung auf focusedIdx", 10, 0, 4, 5, 0, 4},
		{"Fokus außerhalb (negativ) faellt auf Start zurueck", 10, -1, 4, 5, 0, 4},
		{"Liste seit letztem Sync geschrumpft: prevStart reklemmt", 5, 2, 3, 10, 2, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end := anchoredWindow(c.total, c.focused, c.maxVis, c.prevStart)
			if start != c.wantStart || end != c.wantEnd {
				t.Fatalf("erwarte (%d,%d), habe (%d,%d)", c.wantStart, c.wantEnd, start, end)
			}
		})
	}
}

func TestSyncScroll_FocusMovingWithinWindow_KeepsScrollStartStable(t *testing.T) {
	board := domain.Board{}
	for i := 0; i < 10; i++ {
		board = board.AddPrompt(domain.PromptID(fmt.Sprintf("t%d", i)), fmt.Sprintf("Prompt %d", i), 100)
	}
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)

	for i := 0; i < 5; i++ {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
		m = updated.(Model)
	}
	if m.listScrollStart != 0 {
		t.Fatalf("erwarte unveraendertes Scroll-Fenster (start=0) solange Fokus sichtbar bleibt, habe %d", m.listScrollStart)
	}
}

func TestSyncScroll_FocusFirst_MakesFirstPromptVisibleAgain(t *testing.T) {
	board := domain.Board{}
	for i := 0; i < 10; i++ {
		board = board.AddPrompt(domain.PromptID(fmt.Sprintf("t%d", i)), fmt.Sprintf("Prompt %d", i), 100)
	}
	state := application.AppState{Board: board, History: application.NewHistory(50)}
	m := New(state, application.NewExecutor(nullStore{}, nil), fakeClock{now: 100}, &fakeIDGen{})
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 8})
	m = updated.(Model)

	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("G")})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	m = updated.(Model)

	out := m.View()
	if !strings.Contains(out, "Prompt 0") {
		t.Fatalf("erwarte ersten Prompt sichtbar nach 'g' aus gescrolltem Zustand, habe:\n%s", out)
	}
}

func emptyState() application.AppState {
	return application.AppState{Board: domain.Board{}, History: application.NewHistory(50)}
}

func m0Executor() *application.Executor {
	return application.NewExecutor(nullStore{}, nil)
}
