package application

import "qdrover/internal/domain"

type AppState struct {
	Board     domain.Board
	FocusedID *domain.PromptID
	History   *History
}
