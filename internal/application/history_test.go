package application

import (
	"testing"

	"qdrover/internal/domain"
)

func TestHistory_UndoThenRedo_RestoresOriginalPrompts(t *testing.T) {
	h := NewHistory(50)
	before := []domain.Prompt{}
	after := domain.Board{}.AddPrompt("t1", "neu", 100).Prompts

	h.Push(before)

	undone, ok := h.Undo(after)
	if !ok {
		t.Fatal("erwarte erfolgreiches Undo")
	}
	if len(undone) != 0 {
		t.Fatalf("erwarte leere Prompts nach Undo, habe %d", len(undone))
	}

	redone, ok := h.Redo(undone)
	if !ok {
		t.Fatal("erwarte erfolgreiches Redo")
	}
	if len(redone) != 1 {
		t.Fatalf("erwarte 1 Prompt nach Redo, habe %d", len(redone))
	}
}

func TestHistory_Undo_WithoutHistory_ReturnsFalse(t *testing.T) {
	h := NewHistory(50)
	_, ok := h.Undo(nil)
	if ok {
		t.Fatal("erwarte ok=false ohne History")
	}
}

func TestHistory_Push_EnforcesMaxDepth(t *testing.T) {
	h := NewHistory(2)
	h.Push(domain.Board{}.AddPrompt("t1", "a", 1).Prompts)
	h.Push(domain.Board{}.AddPrompt("t2", "b", 2).Prompts)
	h.Push(domain.Board{}.AddPrompt("t3", "c", 3).Prompts)

	current := domain.Board{}.AddPrompt("t4", "d", 4).Prompts
	_, _ = h.Undo(current)
	_, _ = h.Undo(current)
	_, ok := h.Undo(current)
	if ok {
		t.Fatal("erwarte, dass History nach maxDepth=2 Einträgen erschöpft ist")
	}
}
