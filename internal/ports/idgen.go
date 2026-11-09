package ports

import "qdrover/internal/domain"

type IDGenerator interface {
	NewPromptID() domain.PromptID
}
