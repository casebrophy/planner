package notebus

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID         uuid.UUID
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID
	Content    string
	Source     string
	RawInputID  *uuid.UUID
	Unconfirmed bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type NewNote struct {
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID
	Content     string
	Source      string
	RawInputID  *uuid.UUID
	Unconfirmed bool
}

type UpdateNote struct {
	ContextID  *uuid.UUID
	TaskID     *uuid.UUID
	Content    *string
	Source     *string
	RawInputID *uuid.UUID
	Unconfirmed *bool
}
