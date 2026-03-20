package contextbus

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/casebrophy/planner/business/types/contextoutcome"
	"github.com/casebrophy/planner/business/types/debriefstatus"
)

type Context struct {
	ID            uuid.UUID
	Title         string
	Description   string
	Status        Status
	Summary       string
	LastEvent     *time.Time
	LastThreadAt  *time.Time
	DebriefStatus debriefstatus.Status
	Outcome       *contextoutcome.Outcome
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type NewContext struct {
	Title       string
	Description string
}

type UpdateContext struct {
	Title         *string
	Description   *string
	Status        *Status
	Summary       *string
	DebriefStatus *debriefstatus.Status
	Outcome       *contextoutcome.Outcome
}

type Event struct {
	ID        uuid.UUID
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
	CreatedAt time.Time
}

type NewEvent struct {
	ContextID uuid.UUID
	Kind      string
	Content   string
	Metadata  *json.RawMessage
	SourceID  *uuid.UUID
}

type Status int

const (
	Active Status = iota
	Paused
	Closed
)

func (s Status) String() string {
	switch s {
	case Active:
		return "active"
	case Paused:
		return "paused"
	case Closed:
		return "closed"
	default:
		return "unknown"
	}
}

func Parse(s string) (Status, error) {
	switch s {
	case "active":
		return Active, nil
	case "paused":
		return Paused, nil
	case "closed":
		return Closed, nil
	default:
		return 0, ErrInvalidStatus
	}
}

func MustParse(s string) Status {
	status, _ := Parse(s)
	return status
}

var ErrInvalidStatus = StatusError("invalid status")

type StatusError string

func (e StatusError) Error() string {
	return string(e)
}
