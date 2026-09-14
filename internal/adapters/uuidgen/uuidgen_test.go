package uuidgen

import "testing"

func TestGenerator_NewPromptID_ReturnsUniqueNonEmptyIDs(t *testing.T) {
	g := New()
	a := g.NewPromptID()
	b := g.NewPromptID()

	if a == "" || b == "" {
		t.Fatal("erwarte nicht-leere IDs")
	}
	if a == b {
		t.Fatal("erwarte zwei unterschiedliche IDs")
	}
}
