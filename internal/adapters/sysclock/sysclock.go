package sysclock

import (
	"time"

	"qdrover/internal/domain"
)

type Clock struct{}

func New() Clock { return Clock{} }

func (Clock) Now() domain.Timestamp {
	return domain.Timestamp(time.Now().UnixMilli())
}
