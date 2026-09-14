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
	PromptIDs      []domain.PromptID // welche Prompts in Text eingeflossen sind
	PrefixCommands []string          // werden vor Text an die Ziel-Pane gesendet, z.B. ["/clear", "/plan"]
}

func (SendDispatch) isEffect() {}
