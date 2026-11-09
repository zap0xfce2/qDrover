package domain

type PromptPosition int

type Scope string

const (
	ScopeGlobal Scope = "global"
	ScopeLocal  Scope = "local"
)

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
	// Scope: leer = global (Alt-Daten ohne dieses Feld).
	Scope Scope
}

func (t Prompt) IsLocal() bool {
	return t.Scope == ScopeLocal
}

func (t Prompt) IsLive() bool {
	return t.DeletedAt == nil
}
