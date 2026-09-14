package main

import (
	"strings"
	"testing"

	"qdrover/internal/domain"
)

func TestDefaultStateDir_EndsWithExpectedSuffix(t *testing.T) {
	dir, err := defaultStateDir()
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !strings.HasSuffix(dir, "/.local/state/qdrover/sessions") {
		t.Fatalf("erwarte Suffix '/.local/state/qdrover/sessions', habe %q", dir)
	}
}

func TestFirstLivePromptID_ReturnsFirstPromptWhenBoardNonEmpty(t *testing.T) {
	board := domain.Board{}
	board = board.AddPrompt("t1", "erste Idee", 100)
	board = board.AddPrompt("t2", "zweite Idee", 100)

	id := firstLivePromptID(board)
	if id == nil || *id != domain.PromptID("t1") {
		t.Fatalf("erwarte t1, habe %v", id)
	}
}

func TestFirstLivePromptID_ReturnsNilWhenBoardEmpty(t *testing.T) {
	if id := firstLivePromptID(domain.Board{}); id != nil {
		t.Fatalf("erwarte nil bei leerem Board, habe %v", id)
	}
}
