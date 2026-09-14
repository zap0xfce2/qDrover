package domain

type Session struct {
	ID           SessionID
	Name         *string
	OriginCwd    string
	CreatedAt    Timestamp
	LastOpenedAt Timestamp
	// PlanModeActive/ClearModeActive: nil bedeutet "alte Session ohne dieses
	// Feld" -> Default true (siehe IsPlanModeActive/IsClearModeActive), sonst
	// der explizit gespeicherte Wert. Gleiches Optional-Pattern wie Name.
	PlanModeActive  *bool
	ClearModeActive *bool
	SentHistory     []SentHistoryEntry
}

// SentHistoryEntry ist ein Eintrag im persistierten Sendehistory-Log
// (Session.SentHistory) — überlebt das Löschen des zugehörigen Prompts.
type SentHistoryEntry struct {
	SentAt Timestamp
	Text   string
}

// MaxSentHistoryEntries begrenzt Session.SentHistory (FIFO, älteste zuerst raus).
const MaxSentHistoryEntries = 5

func (s Session) IsPlanModeActive() bool {
	return s.PlanModeActive == nil || *s.PlanModeActive
}

func (s Session) IsClearModeActive() bool {
	return s.ClearModeActive == nil || *s.ClearModeActive
}

// RecordSent hängt entry an SentHistory an und kappt bei MaxSentHistoryEntries.
func (s Session) RecordSent(entry SentHistoryEntry) Session {
	history := append(append([]SentHistoryEntry{}, s.SentHistory...), entry)
	if len(history) > MaxSentHistoryEntries {
		history = history[len(history)-MaxSentHistoryEntries:]
	}
	s.SentHistory = history
	return s
}
