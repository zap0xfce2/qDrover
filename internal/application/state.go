package application

import "qdrover/internal/domain"

type AppState struct {
	Board     domain.Board
	FocusedID *domain.PromptID
	History   *History
	// Visible filtert Fokus und Senden auf den aktiven Tab; nil = alle.
	Visible func(domain.Prompt) bool
}
