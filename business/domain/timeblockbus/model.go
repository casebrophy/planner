package timeblockbus

import (
	"time"

	"github.com/google/uuid"
)

type TimeBlock struct {
	ID        uuid.UUID
	TaskID    uuid.UUID
	StartsAt  time.Time
	EndsAt    time.Time
	Confirmed bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type NewTimeBlock struct {
	TaskID   uuid.UUID
	StartsAt time.Time
	EndsAt   time.Time
}

type UpdateTimeBlock struct {
	StartsAt  *time.Time
	EndsAt    *time.Time
	Confirmed *bool
}
