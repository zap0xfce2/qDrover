package application

import (
	"context"
	"fmt"

	"qdrover/internal/domain"
	"qdrover/internal/ports"
)

type Executor struct {
	store ports.Store
	herdr ports.HerdrGateway
}

func NewExecutor(store ports.Store, herdr ports.HerdrGateway) *Executor {
	return &Executor{store: store, herdr: herdr}
}

// Execute liefert das gemergte Board des letzten PersistBoard, sonst nil.
func (e *Executor) Execute(ctx context.Context, effects []Effect) (*domain.Board, error) {
	var merged *domain.Board
	for _, eff := range effects {
		switch ef := eff.(type) {
		case PersistBoard:
			saved, err := e.store.Save(ef.Board)
			if err != nil {
				return nil, fmt.Errorf("board persistieren: %w", err)
			}
			merged = &saved
		case SendDispatch:
			if err := e.herdr.ResolveAndPromptWithPrefix(ctx, ef.Direction, ef.PrefixCommands, ef.Text); err != nil {
				return nil, err
			}
		}
	}
	return merged, nil
}
