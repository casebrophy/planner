package eventdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/eventbus"
)

type eventDB struct {
	ID          uuid.UUID  `db:"event_id"`
	ContextID   *uuid.UUID `db:"context_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Location    *string    `db:"location"`
	StartsAt    time.Time  `db:"starts_at"`
	EndsAt      time.Time  `db:"ends_at"`
	AllDay      bool       `db:"all_day"`
	RawInputID  *uuid.UUID `db:"raw_input_id"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

func toDBEvent(e eventbus.Event) eventDB {
	return eventDB{
		ID:          e.ID,
		ContextID:   e.ContextID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartsAt:    e.StartsAt,
		EndsAt:      e.EndsAt,
		AllDay:      e.AllDay,
		RawInputID:  e.RawInputID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toBusEvent(e eventDB) eventbus.Event {
	return eventbus.Event{
		ID:          e.ID,
		ContextID:   e.ContextID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartsAt:    e.StartsAt,
		EndsAt:      e.EndsAt,
		AllDay:      e.AllDay,
		RawInputID:  e.RawInputID,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func toBusEvents(es []eventDB) []eventbus.Event {
	events := make([]eventbus.Event, len(es))
	for i, e := range es {
		events[i] = toBusEvent(e)
	}
	return events
}
