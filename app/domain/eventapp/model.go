package eventapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/eventbus"
)

type Event struct {
	ID          string  `json:"id"`
	ContextID   *string `json:"contextId,omitempty"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    *string `json:"location,omitempty"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      string  `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
	RawInputID  *string `json:"rawInputId,omitempty"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

func (e Event) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

type NewEvent struct {
	ContextID   *string `json:"contextId"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Location    *string `json:"location"`
	StartsAt    string  `json:"startsAt"`
	EndsAt      string  `json:"endsAt"`
	AllDay      bool    `json:"allDay"`
}

type UpdateEvent struct {
	ContextID   *string `json:"contextId"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Location    *string `json:"location"`
	StartsAt    *string `json:"startsAt"`
	EndsAt      *string `json:"endsAt"`
	AllDay      *bool   `json:"allDay"`
}

func toAppEvent(e eventbus.Event) Event {
	ae := Event{
		ID:          e.ID.String(),
		Title:       e.Title,
		Description: e.Description,
		StartsAt:    e.StartsAt.Format(time.RFC3339),
		EndsAt:      e.EndsAt.Format(time.RFC3339),
		AllDay:      e.AllDay,
		CreatedAt:   e.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   e.UpdatedAt.Format(time.RFC3339),
	}

	if e.ContextID != nil {
		s := e.ContextID.String()
		ae.ContextID = &s
	}
	if e.Location != nil {
		ae.Location = e.Location
	}
	if e.RawInputID != nil {
		s := e.RawInputID.String()
		ae.RawInputID = &s
	}

	return ae
}

func toAppEvents(es []eventbus.Event) []Event {
	events := make([]Event, len(es))
	for i, e := range es {
		events[i] = toAppEvent(e)
	}
	return events
}

func toBusNewEvent(ne NewEvent) (eventbus.NewEvent, error) {
	bne := eventbus.NewEvent{
		Title:       ne.Title,
		Description: ne.Description,
		Location:    ne.Location,
		AllDay:      ne.AllDay,
	}

	if ne.ContextID != nil {
		id, err := uuid.Parse(*ne.ContextID)
		if err != nil {
			return eventbus.NewEvent{}, fmt.Errorf("contextId: %w", err)
		}
		bne.ContextID = &id
	}

	startsAt, err := time.Parse(time.RFC3339, ne.StartsAt)
	if err != nil {
		return eventbus.NewEvent{}, fmt.Errorf("startsAt: %w", err)
	}
	bne.StartsAt = startsAt

	endsAt, err := time.Parse(time.RFC3339, ne.EndsAt)
	if err != nil {
		return eventbus.NewEvent{}, fmt.Errorf("endsAt: %w", err)
	}
	bne.EndsAt = endsAt

	return bne, nil
}

func toBusUpdateEvent(ue UpdateEvent) (eventbus.UpdateEvent, error) {
	var bue eventbus.UpdateEvent

	bue.ContextID = nil
	if ue.ContextID != nil {
		id, err := uuid.Parse(*ue.ContextID)
		if err != nil {
			return eventbus.UpdateEvent{}, fmt.Errorf("contextId: %w", err)
		}
		bue.ContextID = &id
	}

	bue.Title = ue.Title
	bue.Description = ue.Description
	bue.Location = ue.Location
	bue.AllDay = ue.AllDay

	if ue.StartsAt != nil {
		t, err := time.Parse(time.RFC3339, *ue.StartsAt)
		if err != nil {
			return eventbus.UpdateEvent{}, fmt.Errorf("startsAt: %w", err)
		}
		bue.StartsAt = &t
	}

	if ue.EndsAt != nil {
		t, err := time.Parse(time.RFC3339, *ue.EndsAt)
		if err != nil {
			return eventbus.UpdateEvent{}, fmt.Errorf("endsAt: %w", err)
		}
		bue.EndsAt = &t
	}

	return bue, nil
}
