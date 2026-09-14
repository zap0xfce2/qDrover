package application

import (
	"context"
	"fmt"

	"qdrover/internal/ports"
)

type Executor struct {
	store ports.Store
	herdr ports.HerdrGateway
}

func NewExecutor(store ports.Store, herdr ports.HerdrGateway) *Executor {
	return &Executor{store: store, herdr: herdr}
}

func (e *Executor) Execute(ctx context.Context, effects []Effect) error {
	for _, eff := range effects {
		switch ef := eff.(type) {
		case PersistBoard:
			if err := e.store.SaveSession(ef.Board); err != nil {
				return fmt.Errorf("board persistieren: %w", err)
			}
		case SendDispatch:
			if err := e.herdr.ResolveAndPromptWithPrefix(ctx, ef.Direction, ef.PrefixCommands, ef.Text); err != nil {
				return err
			}
		}
	}
	return nil
}
