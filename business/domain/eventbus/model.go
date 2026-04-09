package eventbus

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          uuid.UUID
	ContextID   *uuid.UUID
	Title       string
	Description string
	Location    *string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
	RawInputID  *uuid.UUID
	Unconfirmed bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NewEvent struct {
	ContextID   *uuid.UUID
	Title       string
	Description string
	Location    *string
	StartsAt    time.Time
	EndsAt      time.Time
	AllDay      bool
	RawInputID  *uuid.UUID
	Unconfirmed bool
}

type UpdateEvent struct {
	ContextID   *uuid.UUID
	Title       *string
	Description *string
	Location    *string
	StartsAt    *time.Time
	EndsAt      *time.Time
	AllDay      *bool
}
