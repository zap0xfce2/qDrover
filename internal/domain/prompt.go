package domain

type PromptPosition int

type Prompt struct {
	ID        PromptID
	Content   string
	Position  PromptPosition
	CreatedAt Timestamp
	UpdatedAt Timestamp
	DeletedAt *Timestamp
	// Marked: überstimmt RemoveAfterSend beim Senden mit "s" — der Prompt
	// bleibt erhalten, bis die Markierung erneut umgeschaltet wird.
	Marked bool
}

func (t Prompt) IsLive() bool {
	return t.DeletedAt == nil
}
