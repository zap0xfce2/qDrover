package ports

import "qdrover/internal/domain"

type Clock interface {
	Now() domain.Timestamp
}
