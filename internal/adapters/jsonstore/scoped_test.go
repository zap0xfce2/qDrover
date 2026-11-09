package jsonstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"qdrover/internal/application"
	"qdrover/internal/domain"
)

type scopedPaths struct {
	global string
	local  string
}

func newScopedPaths(t *testing.T) scopedPaths {
	t.Helper()
	dir := t.TempDir()
	return scopedPaths{
		global: filepath.Join(dir, "state", "global.json"),
		local:  filepath.Join(dir, "project", "qdrover.json"),
	}
}

func (p scopedPaths) store() *ScopedStore {
	return NewScopedStore(p.global, p.local)
}

func mustLoad(t *testing.T, s *ScopedStore) domain.Board {
	t.Helper()
	board, err := s.Load()
	if err != nil {
		t.Fatalf("Load fehlgeschlagen: %v", err)
	}
	return board
}

func mustSave(t *testing.T, s *ScopedStore, board domain.Board) domain.Board {
	t.Helper()
	merged, err := s.Save(board)
	if err != nil {
		t.Fatalf("Save fehlgeschlagen: %v", err)
	}
	return merged
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("%s lesen: %v", path, err)
	}
	return data
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("verzeichnis anlegen: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("%s schreiben: %v", path, err)
	}
}

// seedLocal legt einen lokalen Prompt "l1" an und liefert die Bytes der Lokaldatei.
func seedLocal(t *testing.T, p scopedPaths) []byte {
	t.Helper()
	s := p.store()
	board := mustLoad(t, s).AddScopedPrompt("l1", "lokal", domain.ScopeLocal, 100)
	mustSave(t, s, board)
	return mustRead(t, p.local)
}

func findPrompt(board domain.Board, id domain.PromptID) (domain.Prompt, bool) {
	for _, prompt := range board.LivePrompts() {
		if prompt.ID == id {
			return prompt, true
		}
	}
	return domain.Prompt{}, false
}

func TestScopedStore_TwoInstancesBothAddGlobal_BothKept(t *testing.T) {
	p := newScopedPaths(t)
	a, b := p.store(), p.store()
	boardA, boardB := mustLoad(t, a), mustLoad(t, b)

	mustSave(t, a, boardA.AddPrompt("ga", "von a", 100))
	mustSave(t, b, boardB.AddPrompt("gb", "von b", 101))

	if got := len(mustLoad(t, p.store()).LivePrompts()); got != 2 {
		t.Fatalf("erwarte 2 Prompts, habe %d", got)
	}
}

func TestScopedStore_TwoInstancesBothAddLocal_BothKept(t *testing.T) {
	p := newScopedPaths(t)
	a, b := p.store(), p.store()
	boardA, boardB := mustLoad(t, a), mustLoad(t, b)

	mustSave(t, a, boardA.AddScopedPrompt("la", "von a", domain.ScopeLocal, 100))
	mustSave(t, b, boardB.AddScopedPrompt("lb", "von b", domain.ScopeLocal, 101))

	if got := len(mustLoad(t, p.store()).LivePrompts()); got != 2 {
		t.Fatalf("erwarte 2 Prompts, habe %d", got)
	}
}

func TestScopedStore_DeletedByOther_NotResurrected(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)
	a, b := p.store(), p.store()
	boardA, boardB := mustLoad(t, a), mustLoad(t, b)
	deleted, err := boardB.DeletePrompt("l1", 200)
	if err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}
	mustSave(t, b, deleted)

	merged := mustSave(t, a, boardA)

	if _, ok := findPrompt(merged, "l1"); ok {
		t.Fatalf("gelöschter Prompt wurde wiederbelebt: %+v", merged.Prompts)
	}
}

func TestScopedStore_Load_ConflictMarkersInLocal_ErrorNamesPath(t *testing.T) {
	p := newScopedPaths(t)
	mustWrite(t, p.local, "<<<<<<< HEAD\n{\"prompts\": []}\n=======\n{}\n>>>>>>> other\n")

	_, err := p.store().Load()

	if err == nil || !strings.Contains(err.Error(), p.local) {
		t.Fatalf("erwarte Fehler mit Pfad %s, habe %v", p.local, err)
	}
}

func TestScopedStore_Save_CorruptLocal_GlobalNotWritten(t *testing.T) {
	p := newScopedPaths(t)
	s := p.store()
	board := mustLoad(t, s).AddPrompt("g1", "global", 100)
	mustWrite(t, p.local, "kaputt")

	_, _ = s.Save(board)

	if _, err := os.Stat(p.global); !os.IsNotExist(err) {
		t.Fatalf("global.json darf bei Lesefehler nicht geschrieben werden, Stat: %v", err)
	}
}

func TestScopedStore_Load_LocalPromptHasLocalScope(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)

	prompt, _ := findPrompt(mustLoad(t, p.store()), "l1")

	if prompt.Scope != domain.ScopeLocal {
		t.Fatalf("erwarte Scope local, habe %q", prompt.Scope)
	}
}

func TestScopedStore_Load_GlobalBeforeLocal(t *testing.T) {
	p := newScopedPaths(t)
	s := p.store()
	board := mustLoad(t, s).AddScopedPrompt("l1", "lokal", domain.ScopeLocal, 100)
	mustSave(t, s, board.AddPrompt("g1", "global", 101))

	live := mustLoad(t, p.store()).LivePrompts()

	if live[0].ID != "g1" {
		t.Fatalf("erwarte globalen Prompt zuerst, habe %+v", live)
	}
}

func TestScopedStore_ToggleMarkOnLocal_LocalFileUnchanged(t *testing.T) {
	p := newScopedPaths(t)
	before := seedLocal(t, p)
	s := p.store()
	board, err := mustLoad(t, s).ToggleMarked("l1")
	if err != nil {
		t.Fatalf("ToggleMarked: %v", err)
	}

	mustSave(t, s, board)

	if after := mustRead(t, p.local); string(after) != string(before) {
		t.Fatalf("qdrover.json verändert:\nvorher %s\nnachher %s", before, after)
	}
}

func TestScopedStore_PlanModeToggle_LocalFileUnchanged(t *testing.T) {
	p := newScopedPaths(t)
	before := seedLocal(t, p)
	s := p.store()
	board := mustLoad(t, s)
	planOff := false
	board.Session.PlanModeActive = &planOff

	mustSave(t, s, board)

	if after := mustRead(t, p.local); string(after) != string(before) {
		t.Fatalf("qdrover.json verändert:\nvorher %s\nnachher %s", before, after)
	}
}

func TestScopedStore_RecordSent_LocalFileUnchanged(t *testing.T) {
	p := newScopedPaths(t)
	before := seedLocal(t, p)
	s := p.store()
	board := mustLoad(t, s)
	board.Session = board.Session.RecordSent(domain.SentHistoryEntry{SentAt: 300, Text: "lokal"})

	mustSave(t, s, board)

	if after := mustRead(t, p.local); string(after) != string(before) {
		t.Fatalf("qdrover.json verändert:\nvorher %s\nnachher %s", before, after)
	}
}

func TestScopedStore_NoLocalPrompts_NoLocalFileCreated(t *testing.T) {
	p := newScopedPaths(t)
	s := p.store()

	mustSave(t, s, mustLoad(t, s).AddPrompt("g1", "global", 100))

	if _, err := os.Stat(p.local); !os.IsNotExist(err) {
		t.Fatalf("qdrover.json darf nicht angelegt werden, Stat: %v", err)
	}
}

func TestScopedStore_DeleteLastLocal_FileKeptEmpty(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)
	s := p.store()
	board, err := mustLoad(t, s).DeletePrompt("l1", 200)
	if err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}

	mustSave(t, s, board)

	if _, err := os.Stat(p.local); err != nil {
		t.Fatalf("qdrover.json muss erhalten bleiben, Stat: %v", err)
	}
}

func TestScopedStore_LocalFile_HasNoMarkedOrDeletedAt(t *testing.T) {
	p := newScopedPaths(t)
	s := p.store()
	board := mustLoad(t, s).AddScopedPrompt("l1", "markiert", domain.ScopeLocal, 100)
	board = board.AddScopedPrompt("l2", "gelöscht", domain.ScopeLocal, 101)
	board, _ = board.ToggleMarked("l1")
	board, _ = board.DeletePrompt("l2", 200)

	mustSave(t, s, board)

	text := strings.ToLower(string(mustRead(t, p.local)))
	if strings.Contains(text, "marked") || strings.Contains(text, "deletedat") {
		t.Fatalf("qdrover.json enthält persönliche Felder: %s", text)
	}
}

func TestScopedStore_ExternalLocalAdd_AppearsAfterSave(t *testing.T) {
	p := newScopedPaths(t)
	s := p.store()
	board := mustSave(t, s, mustLoad(t, s).AddScopedPrompt("l1", "meins", domain.ScopeLocal, 100))
	mustWrite(t, p.local, `{"prompts": [
		{"id": "l1", "content": "meins", "position": 0, "createdAt": 100, "updatedAt": 100},
		{"id": "fremd", "content": "von git", "position": 1, "createdAt": 150, "updatedAt": 150}
	]}`)

	merged := mustSave(t, s, board)

	if _, ok := findPrompt(merged, "fremd"); !ok {
		t.Fatalf("fremder Prompt fehlt im Board: %+v", merged.Prompts)
	}
}

func TestScopedStore_LoadSave_RoundTripKeepsLocalMarks(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)
	s := p.store()
	board, err := mustLoad(t, s).ToggleMarked("l1")
	if err != nil {
		t.Fatalf("ToggleMarked: %v", err)
	}
	mustSave(t, s, board)

	prompt, _ := findPrompt(mustLoad(t, p.store()), "l1")

	if !prompt.Marked {
		t.Fatalf("Markierung des lokalen Prompts verloren: %+v", prompt)
	}
}

func TestScopedStore_MarkByOtherInstance_Kept(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)
	a, b := p.store(), p.store()
	boardA, boardB := mustLoad(t, a), mustLoad(t, b)
	marked, _ := boardB.ToggleMarked("l1")
	mustSave(t, b, marked)

	merged := mustSave(t, a, boardA.AddPrompt("g1", "global", 300))

	if prompt, _ := findPrompt(merged, "l1"); !prompt.Marked {
		t.Fatalf("Markierung der anderen Instanz verloren: %+v", prompt)
	}
}

func TestScopedStore_MarksOfOtherProjects_Kept(t *testing.T) {
	p := newScopedPaths(t)
	mustWrite(t, p.global, `{"session": {}, "prompts": [], "localMarks": ["anderes-projekt"]}`)
	s := p.store()

	mustSave(t, s, mustLoad(t, s).AddPrompt("g1", "global", 100))

	if !strings.Contains(string(mustRead(t, p.global)), "anderes-projekt") {
		t.Fatalf("Markierung eines anderen Projekts verloren: %s", mustRead(t, p.global))
	}
}

// Die UI behält Soft-Deletes für Undo im Board; ein alter Grabstein darf einen
// extern (git, andere Instanz) wieder angelegten Prompt nicht erneut löschen.
func TestScopedStore_StaleTombstone_DoesNotDeleteExternalReAdd(t *testing.T) {
	p := newScopedPaths(t)
	before := seedLocal(t, p)
	s := p.store()
	withTombstone, err := mustLoad(t, s).DeletePrompt("l1", 200)
	if err != nil {
		t.Fatalf("DeletePrompt: %v", err)
	}
	mustSave(t, s, withTombstone)
	mustWrite(t, p.local, string(before))

	merged := mustSave(t, s, withTombstone)

	if _, ok := findPrompt(merged, "l1"); !ok {
		t.Fatalf("extern wieder angelegter Prompt durch alten Grabstein gelöscht: %+v", merged.Prompts)
	}
}

func TestScopedStore_UnchangedLocal_NotRewritten(t *testing.T) {
	p := newScopedPaths(t)
	seedLocal(t, p)
	past := time.Unix(1_000_000, 0)
	if err := os.Chtimes(p.local, past, past); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}
	s := p.store()
	board := mustLoad(t, s)
	planOff := false
	board.Session.PlanModeActive = &planOff

	mustSave(t, s, board)

	info, err := os.Stat(p.local)
	if err != nil || !info.ModTime().Equal(past) {
		t.Fatalf("qdrover.json neu geschrieben (mtime %v, err %v)", info.ModTime(), err)
	}
}

type countingClock struct{ now domain.Timestamp }

func (c *countingClock) Now() domain.Timestamp { c.now++; return c.now }

type countingIDs struct{ next int }

func (g *countingIDs) NewPromptID() domain.PromptID {
	g.next++
	return domain.PromptID(string(rune('a' + g.next)))
}

func (g *countingIDs) NewSessionID() domain.SessionID { return "s" }

// reduceAndSave spielt den UI-Pfad nach: Reduce, PersistBoard speichern, Merge zurückspielen.
func reduceAndSave(t *testing.T, s *ScopedStore, state application.AppState, action application.Action) application.AppState {
	t.Helper()
	clock, ids := &countingClock{now: 1000}, &countingIDs{}
	state, effects, err := application.Reduce(state, action, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	for _, effect := range effects {
		if persist, ok := effect.(application.PersistBoard); ok {
			merged := mustSave(t, s, persist.Board)
			state, _, _ = application.Reduce(state, application.ApplyMerged{Board: merged}, clock, ids)
		}
	}
	return state
}

func TestScopedStore_UndoAfterForeignAdd_KeepsForeignPrompt(t *testing.T) {
	paths := newScopedPaths(t)
	a, b := paths.store(), paths.store()
	state := application.AppState{Board: mustLoad(t, a), History: application.NewHistory(50)}
	state = reduceAndSave(t, a, state, application.CreatePrompt{Content: "mine"})
	mustSave(t, b, mustLoad(t, b).AddScopedPrompt("foreign", "von B", domain.ScopeGlobal, 50))

	state = reduceAndSave(t, a, state, application.TogglePlanMode{})
	reduceAndSave(t, a, state, application.Undo{})

	if _, ok := findPrompt(mustLoad(t, b), "foreign"); !ok {
		t.Fatalf("fremder Prompt verloren: %+v", mustLoad(t, b).Prompts)
	}
}

func TestScopedStore_MarkAfterForeignEdit_KeepsForeignContent(t *testing.T) {
	paths := newScopedPaths(t)
	a, b := paths.store(), paths.store()
	state := application.AppState{Board: mustLoad(t, a), History: application.NewHistory(50)}
	state = reduceAndSave(t, a, state, application.CreatePrompt{Content: "alt"})
	id := *state.FocusedID
	foreign, _ := mustLoad(t, b).EditPrompt(id, "neu von B", 99)
	mustSave(t, b, foreign)

	reduceAndSave(t, a, state, application.ToggleMarked{ID: id})

	if got := mustLoad(t, b).LivePrompts()[0].Content; got != "neu von B" {
		t.Fatalf("fremde Bearbeitung verloren: %q", got)
	}
}

func TestScopedStore_FailedSendOfLocal_LocalFileUnchanged(t *testing.T) {
	paths := newScopedPaths(t)
	before := seedLocal(t, paths)
	s := paths.store()
	state := application.AppState{Board: mustLoad(t, s), History: application.NewHistory(50)}

	state = reduceAndSave(t, s, state, application.DeletePrompt{ID: "l1"})
	reduceAndSave(t, s, state, application.RestorePrompt{ID: "l1"})

	if after := mustRead(t, paths.local); string(after) != string(before) {
		t.Fatalf("erwarte unveränderte Lokaldatei:\n%s\nhabe:\n%s", before, after)
	}
}
