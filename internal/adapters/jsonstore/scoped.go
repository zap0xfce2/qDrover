package jsonstore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"qdrover/internal/domain"
)

// ScopedStore verteilt das Board auf global.json (globale Prompts + persönlicher
// Zustand) und qdrover.json (nur lokale Projekt-Prompts). Jedes Save liest beide
// Dateien neu und mergt 3-Wege gegen den zuletzt gelesenen/geschriebenen Stand,
// damit parallele Instanzen und git-Änderungen nicht überschrieben werden.
type ScopedStore struct {
	globalPath string
	localPath  string
	baseGlobal globalState
	baseLocal  []domain.Prompt
}

func NewScopedStore(globalPath, localPath string) *ScopedStore {
	return &ScopedStore{globalPath: globalPath, localPath: localPath}
}

type globalFile struct {
	Session    domain.Session    `json:"session"`
	Prompts    []domain.Prompt   `json:"prompts"`
	LocalMarks []domain.PromptID `json:"localMarks"`
}

// localPrompt lässt Marked/DeletedAt/Scope bewusst weg: persönlicher Zustand
// gehört nicht in eine potenziell eingecheckte Projektdatei.
type localPrompt struct {
	ID        domain.PromptID       `json:"id"`
	Content   string                `json:"content"`
	Position  domain.PromptPosition `json:"position"`
	CreatedAt domain.Timestamp      `json:"createdAt"`
	UpdatedAt domain.Timestamp      `json:"updatedAt"`
}

type localFile struct {
	Prompts []localPrompt `json:"prompts"`
}

// globalState ist global.json in Arbeitsform; zugleich Merge-Basis.
type globalState struct {
	session    domain.Session
	prompts    []domain.Prompt
	localMarks map[domain.PromptID]bool
}

func (s *ScopedStore) Load() (domain.Board, error) {
	global, _, err := readGlobal(s.globalPath)
	if err != nil {
		return domain.Board{}, err
	}
	local, _, err := readLocal(s.localPath)
	if err != nil {
		return domain.Board{}, err
	}
	s.baseGlobal, s.baseLocal = global, local
	return composeBoard(global, local), nil
}

func (s *ScopedStore) Save(board domain.Board) (domain.Board, error) {
	theirsGlobal, globalOnDisk, err := readGlobal(s.globalPath)
	if err != nil {
		return board, err
	}
	theirsLocal, localOnDisk, err := readLocal(s.localPath)
	if err != nil {
		return board, err
	}
	mineGlobal, mineLocal := splitBoard(board, s.baseGlobal.localMarks)
	// ponytail: kein Datei-Lock zwischen Lesen und Schreiben; ein exakt
	// gleichzeitiges Save einer anderen Instanz kann in diesem Fenster verloren
	// gehen. flock/Lockdatei nachrüsten, falls das real auftritt.
	mergedGlobal := mergeGlobal(s.baseGlobal, mineGlobal, theirsGlobal)
	mergedLocal := domain.MergePrompts(s.baseLocal, mineLocal, theirsLocal)
	if err := writeGlobal(s.globalPath, mergedGlobal, globalOnDisk); err != nil {
		return board, err
	}
	s.baseGlobal = mergedGlobal
	if err := writeLocal(s.localPath, mergedLocal, localOnDisk); err != nil {
		return board, err
	}
	s.baseLocal = mergedLocal
	return composeBoard(mergedGlobal, mergedLocal), nil
}

// readJSON liefert die Rohbytes (nil = Datei fehlt) für den späteren Byte-Vergleich.
func readJSON(path string, into any) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("%s lesen: %w", path, err)
	}
	if isBlank(raw) {
		// z. B. per touch angelegt: gilt als leere Datei, nicht als Parse-Fehler.
		return raw, nil
	}
	if err := json.Unmarshal(raw, into); err != nil {
		return nil, fmt.Errorf("%s parsen: %w", path, err)
	}
	return raw, nil
}

func isBlank(raw []byte) bool {
	return len(bytes.TrimSpace(raw)) == 0
}

func readGlobal(path string) (globalState, []byte, error) {
	var file globalFile
	raw, err := readJSON(path, &file)
	if err != nil {
		return globalState{}, nil, err
	}
	prompts := livePrompts(file.Prompts)
	for i := range prompts {
		prompts[i].Scope = domain.ScopeGlobal
	}
	marks := map[domain.PromptID]bool{}
	for _, id := range file.LocalMarks {
		marks[id] = true
	}
	return globalState{session: file.Session, prompts: prompts, localMarks: marks}, raw, nil
}

func readLocal(path string) ([]domain.Prompt, []byte, error) {
	var file localFile
	raw, err := readJSON(path, &file)
	if err != nil {
		return nil, nil, err
	}
	prompts := make([]domain.Prompt, 0, len(file.Prompts))
	for _, p := range file.Prompts {
		prompts = append(prompts, domain.Prompt{
			ID: p.ID, Content: p.Content, Position: p.Position,
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, Scope: domain.ScopeLocal,
		})
	}
	return livePrompts(prompts), raw, nil
}

// splitBoard teilt das Board in die Arbeitsform beider Dateien. Markierungen
// lokaler Prompts wandern nach localMarks; IDs, die nicht auf diesem Board
// liegen (z. B. andere Projekte), bleiben wie in der Basis, gelten also als
// von mir unverändert.
func splitBoard(board domain.Board, baseMarks map[domain.PromptID]bool) (globalState, []domain.Prompt) {
	mine := globalState{session: board.Session, prompts: []domain.Prompt{}, localMarks: copyMarks(baseMarks)}
	local := []domain.Prompt{}
	for _, p := range board.Prompts {
		if !p.IsLocal() {
			p.Scope = domain.ScopeGlobal
			mine.prompts = append(mine.prompts, p)
			continue
		}
		mine.localMarks[p.ID] = p.IsLive() && p.Marked
		p.Marked = false
		local = append(local, p)
	}
	return mine, local
}

func copyMarks(marks map[domain.PromptID]bool) map[domain.PromptID]bool {
	copied := make(map[domain.PromptID]bool, len(marks))
	for id, marked := range marks {
		copied[id] = marked
	}
	return copied
}

func mergeGlobal(base, mine, theirs globalState) globalState {
	return globalState{
		session:    domain.MergeSession(base.session, mine.session, theirs.session),
		prompts:    domain.MergePrompts(base.prompts, mine.prompts, theirs.prompts),
		localMarks: mergeMarks(base.localMarks, mine.localMarks, theirs.localMarks),
	}
}

// mergeMarks: pro ID gewinnt mine, wenn ich die Mitgliedschaft geändert habe, sonst theirs.
func mergeMarks(base, mine, theirs map[domain.PromptID]bool) map[domain.PromptID]bool {
	merged := map[domain.PromptID]bool{}
	for _, index := range []map[domain.PromptID]bool{base, mine, theirs} {
		for id := range index {
			marked := theirs[id]
			if mine[id] != base[id] {
				marked = mine[id]
			}
			if marked {
				merged[id] = true
			}
		}
	}
	return merged
}

func writeGlobal(path string, state globalState, onDisk []byte) error {
	marks := make([]domain.PromptID, 0, len(state.localMarks))
	for id := range state.localMarks {
		marks = append(marks, id)
	}
	sort.Slice(marks, func(i, j int) bool { return marks[i] < marks[j] })
	return writeJSONIfChanged(path, globalFile{Session: state.session, Prompts: state.prompts, LocalMarks: marks}, onDisk)
}

// writeLocal legt qdrover.json nur an, wenn es lokale Prompts gibt; eine
// bestehende Datei wird auch leer weitergeführt, eine inhaltsleere bleibt unberührt.
func writeLocal(path string, prompts []domain.Prompt, onDisk []byte) error {
	if isBlank(onDisk) && len(prompts) == 0 {
		return nil
	}
	file := localFile{Prompts: make([]localPrompt, 0, len(prompts))}
	for i, p := range livePrompts(prompts) {
		file.Prompts = append(file.Prompts, localPrompt{
			ID: p.ID, Content: p.Content, Position: domain.PromptPosition(i),
			CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt,
		})
	}
	return writeJSONIfChanged(path, file, onDisk)
}

// writeJSONIfChanged schreibt nur bei geänderten Bytes (schont git-Diffs und mtime).
func writeJSONIfChanged(path string, content any, onDisk []byte) error {
	data, err := json.MarshalIndent(content, "", "  ")
	if err != nil {
		return fmt.Errorf("%s serialisieren: %w", path, err)
	}
	data = append(data, '\n')
	if onDisk != nil && bytes.Equal(data, onDisk) {
		return nil
	}
	return writeAtomic(path, data)
}

// writeAtomic: eindeutige Temp-Datei je Aufruf, damit parallele Instanzen sich nicht in die Quere kommen.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("%s verzeichnis anlegen: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("%s temp-datei anlegen: %w", path, err)
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("%s temp-datei schreiben: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("%s temp-datei schließen: %w", path, err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("%s temp-datei umbenennen: %w", path, err)
	}
	return nil
}

// composeBoard: globale Prompts zuerst, dann lokale (mit Markierungen), Positionen dicht neu.
func composeBoard(global globalState, local []domain.Prompt) domain.Board {
	locals := livePrompts(local)
	for i := range locals {
		locals[i].Marked = global.localMarks[locals[i].ID]
	}
	prompts := append(livePrompts(global.prompts), locals...)
	for i := range prompts {
		prompts[i].Position = domain.PromptPosition(i)
	}
	return domain.Board{Session: global.session, Prompts: prompts}
}

// livePrompts liefert eine nach Position sortierte Kopie ohne gelöschte Prompts.
func livePrompts(prompts []domain.Prompt) []domain.Prompt {
	return domain.Board{Prompts: prompts}.LivePrompts()
}
