package emailbus

import (
	"time"

	"github.com/google/uuid"
)

type Email struct {
	ID          uuid.UUID
	RawInputID  uuid.UUID
	MessageID   *string
	FromAddress string
	FromName    *string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    *string
	ReceivedAt  time.Time
	ContextID   *uuid.UUID
	CreatedAt   time.Time
}

type NewEmail struct {
	RawInputID  uuid.UUID
	MessageID   *string
	FromAddress string
	FromName    *string
	ToAddress   string
	Subject     string
	BodyText    string
	BodyHTML    *string
	ReceivedAt  time.Time
	ContextID   *uuid.UUID
}

type UpdateEmail struct {
	ContextID *uuid.UUID
}
