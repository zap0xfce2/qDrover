package jsonstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"qdrover/internal/domain"
)

type FileStore struct {
	dir string
}

func NewFileStore(dir string) (*FileStore, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("state-verzeichnis anlegen: %w", err)
	}
	return &FileStore{dir: dir}, nil
}

type fileFormat struct {
	Session domain.Session  `json:"session"`
	Prompts []domain.Prompt `json:"prompts"`
}

func (s *FileStore) sessionPath(id domain.SessionID) string {
	return filepath.Join(s.dir, string(id)+".json")
}

func (s *FileStore) SaveSession(board domain.Board) error {
	data, err := json.MarshalIndent(fileFormat{Session: board.Session, Prompts: board.Prompts}, "", "  ")
	if err != nil {
		return fmt.Errorf("board serialisieren: %w", err)
	}
	path := s.sessionPath(board.Session.ID)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("temp-datei schreiben: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("temp-datei umbenennen: %w", err)
	}
	return nil
}

func (s *FileStore) LoadSession(id domain.SessionID) (domain.Board, error) {
	data, err := os.ReadFile(s.sessionPath(id))
	if err != nil {
		return domain.Board{}, fmt.Errorf("session lesen: %w", err)
	}
	var ff fileFormat
	if err := json.Unmarshal(data, &ff); err != nil {
		return domain.Board{}, fmt.Errorf("session parsen: %w", err)
	}
	return domain.Board{Session: ff.Session, Prompts: ff.Prompts}, nil
}

func (s *FileStore) ListSessions() ([]domain.Session, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("state-verzeichnis lesen: %w", err)
	}
	var sessions []domain.Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		id := domain.SessionID(strings.TrimSuffix(e.Name(), ".json"))
		board, err := s.LoadSession(id)
		if err != nil {
			fmt.Fprintf(os.Stderr, "qdrover: session %s konnte nicht geladen werden: %v\n", id, err)
			continue
		}
		sessions = append(sessions, board.Session)
	}
	return sessions, nil
}
