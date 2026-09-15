package domain

import "testing"

func TestAddPrompt_AppendsAtNextPosition(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "erste Idee", 100)
	b = b.AddPrompt("t2", "zweite Idee", 200)

	live := b.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte 2 lebende Prompts, habe %d", len(live))
	}
	if live[0].Position != 0 || live[1].Position != 1 {
		t.Fatalf("erwarte dichte Positionen 0,1, habe %d,%d", live[0].Position, live[1].Position)
	}
}

func TestDeletePrompt_ReindexesRemainingLivePositions(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "a", 100)
	b = b.AddPrompt("t2", "b", 100)
	b = b.AddPrompt("t3", "c", 100)

	b, err := b.DeletePrompt("t2", 200)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := b.LivePrompts()
	if len(live) != 2 {
		t.Fatalf("erwarte 2 lebende Prompts, habe %d", len(live))
	}
	if live[0].ID != "t1" || live[0].Position != 0 {
		t.Fatalf("erwarte t1 an Position 0, habe %s an %d", live[0].ID, live[0].Position)
	}
	if live[1].ID != "t3" || live[1].Position != 1 {
		t.Fatalf("erwarte t3 an Position 1, habe %s an %d", live[1].ID, live[1].Position)
	}
}

func TestMovePrompt_ReordersLivePrompts(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "a", 100)
	b = b.AddPrompt("t2", "b", 100)
	b = b.AddPrompt("t3", "c", 100)

	b, err := b.MovePrompt("t3", 0, 200)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := b.LivePrompts()
	if live[0].ID != "t3" {
		t.Fatalf("erwarte t3 an erster Stelle, habe %s", live[0].ID)
	}
}

func TestToggleMarked_TogglesPromptMarkedField(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "a", 100)

	b, err := b.ToggleMarked("t1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if !b.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked=true nach erstem ToggleMarked")
	}

	b, err = b.ToggleMarked("t1")
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}
	if b.LivePrompts()[0].Marked {
		t.Fatal("erwarte Marked=false nach zweitem ToggleMarked")
	}
}

func TestToggleMarked_ErrorsForUnknownPrompt(t *testing.T) {
	b := Board{}

	_, err := b.ToggleMarked("unbekannt")
	if err == nil {
		t.Fatal("erwarte Fehler für unbekannte ID")
	}
}

func TestRestorePrompt_ReinsertsAtOriginalPosition(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "a", 100)
	b = b.AddPrompt("t2", "b", 100)
	b = b.AddPrompt("t3", "c", 100)

	b, err := b.DeletePrompt("t2", 200)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	b, err = b.RestorePrompt("t2", 300)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := b.LivePrompts()
	if len(live) != 3 {
		t.Fatalf("erwarte 3 lebende Prompts, habe %d", len(live))
	}
	wantOrder := []PromptID{"t1", "t2", "t3"}
	for i, id := range wantOrder {
		if live[i].ID != id || live[i].Position != PromptPosition(i) {
			t.Fatalf("erwarte Reihenfolge %v mit dichten Positionen, habe %+v", wantOrder, live)
		}
	}
}

func TestRestorePrompt_ErrorsIfNotDeleted(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "a", 100)

	if _, err := b.RestorePrompt("t1", 200); err == nil {
		t.Fatal("erwarte Fehler beim Restore eines noch lebenden Prompts")
	}
	if _, err := b.RestorePrompt("unbekannt", 200); err == nil {
		t.Fatal("erwarte Fehler beim Restore einer unbekannten ID")
	}
}

func TestEditPrompt_UpdatesContentAndTimestamp(t *testing.T) {
	b := Board{}
	b = b.AddPrompt("t1", "alt", 100)

	b, err := b.EditPrompt("t1", "neu", 200)
	if err != nil {
		t.Fatalf("unerwarteter Fehler: %v", err)
	}

	live := b.LivePrompts()
	if live[0].Content != "neu" {
		t.Fatalf("erwarte Content 'neu', habe %q", live[0].Content)
	}
}
