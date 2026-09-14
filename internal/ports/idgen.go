package ports

import "qdrover/internal/domain"

type IDGenerator interface {
	NewPromptID() domain.PromptID
	NewSessionID() domain.SessionID
}
