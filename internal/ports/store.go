package ports

import "qdrover/internal/domain"

type Store interface {
	// Save persistiert das Board und liefert den gemergten Stand zurück.
	Save(board domain.Board) (domain.Board, error)
}
