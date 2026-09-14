package uuidgen

import (
	"github.com/google/uuid"

	"qdrover/internal/domain"
)

type Generator struct{}

func New() Generator { return Generator{} }

func (Generator) NewPromptID() domain.PromptID {
	return domain.PromptID(uuid.Must(uuid.NewV7()).String())
}

func (Generator) NewSessionID() domain.SessionID {
	return domain.SessionID(uuid.Must(uuid.NewV7()).String())
}
