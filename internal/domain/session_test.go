package domain

import "testing"

func TestSession_IsPlanModeActive_NilDefaultsToTrue(t *testing.T) {
	s := Session{}
	if !s.IsPlanModeActive() {
		t.Fatal("erwarte true als Default für PlanModeActive=nil (alte Session ohne dieses Feld)")
	}
}

func TestSession_IsPlanModeActive_RespectsExplicitFalse(t *testing.T) {
	v := false
	s := Session{PlanModeActive: &v}
	if s.IsPlanModeActive() {
		t.Fatal("erwarte false bei explizit gesetztem PlanModeActive=false")
	}
}

func TestSession_IsClearModeActive_NilDefaultsToTrue(t *testing.T) {
	s := Session{}
	if !s.IsClearModeActive() {
		t.Fatal("erwarte true als Default für ClearModeActive=nil (alte Session ohne dieses Feld)")
	}
}

func TestSession_IsClearModeActive_RespectsExplicitFalse(t *testing.T) {
	v := false
	s := Session{ClearModeActive: &v}
	if s.IsClearModeActive() {
		t.Fatal("erwarte false bei explizit gesetztem ClearModeActive=false")
	}
}

func TestSession_RecordSent_AppendsEntry(t *testing.T) {
	s := Session{}
	s = s.RecordSent(SentHistoryEntry{SentAt: 100, Text: "erste Idee"})

	if len(s.SentHistory) != 1 || s.SentHistory[0].Text != "erste Idee" {
		t.Fatalf("erwarte 1 Eintrag 'erste Idee', habe %+v", s.SentHistory)
	}
}

func TestSession_RecordSent_EnforcesMaxEntries(t *testing.T) {
	s := Session{}
	for i := 0; i < MaxSentHistoryEntries+2; i++ {
		s = s.RecordSent(SentHistoryEntry{SentAt: Timestamp(i), Text: "eintrag"})
	}

	if len(s.SentHistory) != MaxSentHistoryEntries {
		t.Fatalf("erwarte %d Einträge (Cap), habe %d", MaxSentHistoryEntries, len(s.SentHistory))
	}
	oldest := s.SentHistory[0]
	if oldest.SentAt != Timestamp(2) {
		t.Fatalf("erwarte älteste verbleibende Zeitstempel=2 (0,1 verdrängt), habe %d", oldest.SentAt)
	}
}
