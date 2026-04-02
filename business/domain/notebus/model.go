package notebus

import (
	"time"

	"github.com/google/uuid"
)

type Note struct {
	ID         uuid.UUID
	ContextID  *uuid.UUID
	Content    string
	Source     string
	RawInputID *uuid.UUID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type NewNote struct {
	ContextID  *uuid.UUID
	Content    string
	Source     string
	RawInputID *uuid.UUID
}

type UpdateNote struct {
	ContextID *uuid.UUID
	Content   *string
	Source    *string
}
