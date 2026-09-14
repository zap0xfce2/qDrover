package ports

import "qdrover/internal/domain"

type Store interface {
	LoadSession(id domain.SessionID) (domain.Board, error)
	SaveSession(board domain.Board) error
	ListSessions() ([]domain.Session, error)
}
