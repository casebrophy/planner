package contextdb

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
)

type contextDB struct {
	ID          uuid.UUID  `db:"context_id"`
	Title       string     `db:"title"`
	Description string     `db:"description"`
	Status      string     `db:"status"`
	Summary     string     `db:"summary"`
	LastEvent   *time.Time `db:"last_event"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
}

type eventDB struct {
	ID        uuid.UUID        `db:"event_id"`
	ContextID uuid.UUID        `db:"context_id"`
	Kind      string           `db:"kind"`
	Content   string           `db:"content"`
	Metadata  *json.RawMessage `db:"metadata"`
	SourceID  *uuid.UUID       `db:"source_id"`
	CreatedAt time.Time        `db:"created_at"`
}

func toDBContext(c contextbus.Context) contextDB {
	return contextDB{
		ID:          c.ID,
		Title:       c.Title,
		Description: c.Description,
		Status:      c.Status.String(),
		Summary:     c.Summary,
		LastEvent:   c.LastEvent,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toBusContext(c contextDB) contextbus.Context {
	return contextbus.Context{
		ID:          c.ID,
		Title:       c.Title,
		Description: c.Description,
		Status:      contextbus.MustParse(c.Status),
		Summary:     c.Summary,
		LastEvent:   c.LastEvent,
		CreatedAt:   c.CreatedAt,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toBusContexts(cs []contextDB) []contextbus.Context {
	result := make([]contextbus.Context, len(cs))
	for i, c := range cs {
		result[i] = toBusContext(c)
	}
	return result
}

func toDBEvent(e contextbus.Event) eventDB {
	return eventDB{
		ID:        e.ID,
		ContextID: e.ContextID,
		Kind:      e.Kind,
		Content:   e.Content,
		Metadata:  e.Metadata,
		SourceID:  e.SourceID,
		CreatedAt: e.CreatedAt,
	}
}

func toBusEvent(e eventDB) contextbus.Event {
	return contextbus.Event{
		ID:        e.ID,
		ContextID: e.ContextID,
		Kind:      e.Kind,
		Content:   e.Content,
		Metadata:  e.Metadata,
		SourceID:  e.SourceID,
		CreatedAt: e.CreatedAt,
	}
}

func toBusEvents(es []eventDB) []contextbus.Event {
	result := make([]contextbus.Event, len(es))
	for i, e := range es {
		result[i] = toBusEvent(e)
	}
	return result
}
