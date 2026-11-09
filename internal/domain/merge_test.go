package domain

import (
	"reflect"
	"testing"
)

func mk(id PromptID, content string) Prompt {
	return Prompt{ID: id, Content: content}
}

func ids(prompts []Prompt) []PromptID {
	result := []PromptID{}
	for _, p := range prompts {
		result = append(result, p.ID)
	}
	return result
}

func TestMergePrompts_TheirNewPromptIsKept(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "")}, []Prompt{mk("a", "")}, []Prompt{mk("a", ""), mk("b", "")})

	if want := []PromptID{"a", "b"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("erwarte %v, habe %v", want, ids(got))
	}
}

func TestMergePrompts_MyNewPromptIsKept(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "")}, []Prompt{mk("a", ""), mk("c", "")}, []Prompt{mk("a", "")})

	if want := []PromptID{"a", "c"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("erwarte %v, habe %v", want, ids(got))
	}
}

func TestMergePrompts_BothAddDifferentPrompts(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "")}, []Prompt{mk("a", ""), mk("c", "")}, []Prompt{mk("a", ""), mk("b", "")})

	if want := []PromptID{"a", "b", "c"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("erwarte %v, habe %v", want, ids(got))
	}
}

func TestMergePrompts_MyEditWins(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{mk("a", "y")}, []Prompt{mk("a", "z")})

	if got[0].Content != "y" {
		t.Fatalf("erwarte y, habe %q", got[0].Content)
	}
}

func TestMergePrompts_TheirEditWinsWhenIUnchanged(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{mk("a", "x")}, []Prompt{mk("a", "z")})

	if got[0].Content != "z" {
		t.Fatalf("erwarte z, habe %q", got[0].Content)
	}
}

func TestMergePrompts_TheirDeleteWinsWhenIUnchanged(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "")}, []Prompt{mk("a", "")}, nil)

	if len(got) != 0 {
		t.Fatalf("erwarte leer, habe %v", ids(got))
	}
}

func TestMergePrompts_MySoftDeleteWins(t *testing.T) {
	deletedAt := Timestamp(5)
	deleted := mk("a", "")
	deleted.DeletedAt = &deletedAt

	got := MergePrompts([]Prompt{mk("a", "")}, []Prompt{deleted}, []Prompt{mk("a", "")})

	if len(got) != 0 {
		t.Fatalf("erwarte leer, habe %v", ids(got))
	}
}

func TestMergeSession_SentHistoryIsUnionCappedAtFive(t *testing.T) {
	mine := Session{SentHistory: []SentHistoryEntry{{1, "a"}, {3, "c"}, {5, "e"}, {7, "g"}}}
	theirs := Session{SentHistory: []SentHistoryEntry{{2, "b"}, {4, "d"}, {6, "f"}}}

	got := MergeSession(Session{}, mine, theirs).SentHistory

	want := []SentHistoryEntry{{3, "c"}, {4, "d"}, {5, "e"}, {6, "f"}, {7, "g"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("erwarte %v, habe %v", want, got)
	}
}

func TestMergeSession_TheirPlanModeWinsWhenIUnchanged(t *testing.T) {
	off := false
	got := MergeSession(Session{}, Session{}, Session{PlanModeActive: &off})

	if got.IsPlanModeActive() {
		t.Fatal("erwarte Plan-Modus aus (theirs)")
	}
}
