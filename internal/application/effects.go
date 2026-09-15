package application

import (
	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

type Effect interface{ isEffect() }

type PersistBoard struct{ Board domain.Board }

func (PersistBoard) isEffect() {}

type SendDispatch struct {
	Direction      ports.Direction
	Text           string
	PromptID       domain.PromptID // welcher Prompt in Text eingeflossen ist
	PrefixCommands []string          // werden vor Text an die Ziel-Pane gesendet, z.B. ["/clear", "/plan"]
}

func (SendDispatch) isEffect() {}
