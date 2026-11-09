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

// Ein Soft-Delete, den die Basis nicht mehr kennt, ist ein alter Grabstein
// (UI behält ihn für Undo) — er darf einen wieder angelegten Prompt nicht löschen.
func TestMergePrompts_StaleTombstoneKeepsTheirPrompt(t *testing.T) {
	deletedAt := Timestamp(5)
	tombstone := mk("a", "")
	tombstone.DeletedAt = &deletedAt

	got := MergePrompts(nil, []Prompt{tombstone}, []Prompt{mk("a", "")})

	if want := []PromptID{"a"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("erwarte %v, habe %v", want, ids(got))
	}
}

func marked(p Prompt) Prompt {
	p.Marked = true
	return p
}

func TestMergePrompts_MyMarkKeepsTheirEdit(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{marked(mk("a", "x"))}, []Prompt{mk("a", "z")})

	if got[0].Content != "z" {
		t.Fatalf("erwarte z, habe %q", got[0].Content)
	}
}

func TestMergePrompts_MyMarkWinsAgainstTheirEdit(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{marked(mk("a", "x"))}, []Prompt{mk("a", "z")})

	if !got[0].Marked {
		t.Fatal("erwarte eigene Markierung")
	}
}

func TestMergePrompts_MyEditKeepsTheirMark(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{mk("a", "y")}, []Prompt{marked(mk("a", "x"))})

	if !got[0].Marked {
		t.Fatal("erwarte fremde Markierung")
	}
}

func TestMergePrompts_MyEditTakesMyUpdatedAt(t *testing.T) {
	mine := mk("a", "y")
	mine.UpdatedAt = 7
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{mine}, []Prompt{marked(mk("a", "x"))})

	if got[0].UpdatedAt != 7 {
		t.Fatalf("erwarte UpdatedAt 7, habe %d", got[0].UpdatedAt)
	}
}

func TestMergePrompts_MyEditResurrectsTheirDelete(t *testing.T) {
	got := MergePrompts([]Prompt{mk("a", "x")}, []Prompt{mk("a", "y")}, []Prompt{})

	if want := []PromptID{"a"}; !reflect.DeepEqual(ids(got), want) {
		t.Fatalf("erwarte %v, habe %v", want, ids(got))
	}
}

func TestMergeSession_SentHistoryIsUnionCappedAtMax(t *testing.T) {
	var mine, theirs Session
	for i := 1; i <= MaxSentHistoryEntries+2; i++ {
		entry := SentHistoryEntry{SentAt: Timestamp(i), Text: "x"}
		if i%2 == 0 {
			theirs.SentHistory = append(theirs.SentHistory, entry)
		} else {
			mine.SentHistory = append(mine.SentHistory, entry)
		}
	}

	got := MergeSession(Session{}, mine, theirs).SentHistory

	if got[0].SentAt != 3 || len(got) != MaxSentHistoryEntries {
		t.Fatalf("erwarte die %d neuesten Einträge ab Zeitstempel 3, habe %v", MaxSentHistoryEntries, got)
	}
}

func TestMergeSession_TheirPlanModeWinsWhenIUnchanged(t *testing.T) {
	off := false
	got := MergeSession(Session{}, Session{}, Session{PlanModeActive: &off})

	if got.IsPlanModeActive() {
		t.Fatal("erwarte Plan-Modus aus (theirs)")
	}
}
