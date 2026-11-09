package application

import (
	"testing"

	"qdrover/internal/domain"
)

func sessionBoard(created, opened domain.Timestamp, contents ...string) domain.Board {
	board := domain.Board{Session: domain.Session{CreatedAt: created, LastOpenedAt: opened}}
	for i, content := range contents {
		board = board.AddPrompt(domain.PromptID(content), content, created+domain.Timestamp(i))
	}
	return board
}

func contentsOf(board domain.Board) []string {
	var contents []string
	for _, p := range board.LivePrompts() {
		contents = append(contents, p.Content)
	}
	return contents
}

func TestMigrateSessions_MergesAllLivePromptsOldestSessionFirst(t *testing.T) {
	newer := sessionBoard(200, 0, "c", "d")
	older := sessionBoard(100, 0, "a", "b")

	got := contentsOf(MigrateSessions([]domain.Board{newer, older}))

	if len(got) != 4 || got[0] != "a" || got[1] != "b" || got[2] != "c" || got[3] != "d" {
		t.Fatalf("erwarte [a b c d], habe %v", got)
	}
}

func TestMigrateSessions_DropsExactDuplicates(t *testing.T) {
	first := sessionBoard(100, 0, "gleich", "nur-a")
	second := sessionBoard(200, 0, "gleich")

	got := contentsOf(MigrateSessions([]domain.Board{first, second}))

	if len(got) != 2 {
		t.Fatalf("erwarte 2 Prompts, habe %v", got)
	}
}

func TestMigrateSessions_KeepsMarked(t *testing.T) {
	board := sessionBoard(100, 0, "a")
	board.Prompts[0].Marked = true

	merged := MigrateSessions([]domain.Board{board})

	if !merged.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked übernommen")
	}
}

func TestMigrateSessions_MakesPromptsGlobalWithDensePositions(t *testing.T) {
	merged := MigrateSessions([]domain.Board{sessionBoard(100, 0, "a"), sessionBoard(200, 0, "b")})

	if p := merged.LivePrompts()[1]; p.Scope != domain.ScopeGlobal || p.Position != 1 {
		t.Fatalf("erwarte global/Position 1, habe %q/%d", p.Scope, p.Position)
	}
}

func TestMigrateSessions_TakesModesFromLastOpenedSession(t *testing.T) {
	off := false
	recent := sessionBoard(100, 900)
	recent.Session.ID = "recent"
	recent.Session.PlanModeActive = &off
	other := sessionBoard(200, 500)
	other.Session.ID = "other"

	merged := MigrateSessions([]domain.Board{recent, other})

	if merged.Session.ID != "recent" || merged.Session.IsPlanModeActive() {
		t.Fatalf("erwarte Session 'recent' mit Plan aus, habe %q", merged.Session.ID)
	}
}

func TestMigrateSessions_SkipsDeletedPrompts(t *testing.T) {
	board := sessionBoard(100, 0, "weg", "bleibt")
	deletedAt := domain.Timestamp(150)
	board.Prompts[0].DeletedAt = &deletedAt

	got := contentsOf(MigrateSessions([]domain.Board{board}))

	if len(got) != 1 || got[0] != "bleibt" {
		t.Fatalf("erwarte [bleibt], habe %v", got)
	}
}
