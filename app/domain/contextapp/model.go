package contextapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/types/contextoutcome"
	"github.com/casebrophy/planner/business/types/debriefstatus"
)

type Context struct {
	ID            string  `json:"id"`
	Title         string  `json:"title"`
	Description   string  `json:"description"`
	Status        string  `json:"status"`
	Summary       string  `json:"summary"`
	LastEvent     *string `json:"lastEvent,omitempty"`
	LastThreadAt  *string `json:"lastThreadAt,omitempty"`
	DebriefStatus string  `json:"debriefStatus"`
	Outcome       *string `json:"outcome,omitempty"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     string  `json:"updatedAt"`
}

func (c Context) Encode() ([]byte, string, error) {
	data, err := json.Marshal(c)
	return data, "application/json", err
}

type Event struct {
	ID        string          `json:"id"`
	ContextID string          `json:"contextId"`
	Kind      string          `json:"kind"`
	Content   string          `json:"content"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	SourceID  *string         `json:"sourceId,omitempty"`
	CreatedAt string          `json:"createdAt"`
}

func (e Event) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

type NewContext struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

type UpdateContext struct {
	Title         *string `json:"title"`
	Description   *string `json:"description"`
	Status        *string `json:"status"`
	Summary       *string `json:"summary"`
	DebriefStatus *string `json:"debriefStatus"`
	Outcome       *string `json:"outcome"`
}

type NewEvent struct {
	Kind     string          `json:"kind"`
	Content  string          `json:"content"`
	Metadata json.RawMessage `json:"metadata,omitempty"`
	SourceID *string         `json:"sourceId"`
}

func toAppContext(c contextbus.Context) Context {
	ac := Context{
		ID:            c.ID.String(),
		Title:         c.Title,
		Description:   c.Description,
		Status:        c.Status.String(),
		Summary:       c.Summary,
		DebriefStatus: c.DebriefStatus.String(),
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:     c.UpdatedAt.Format(time.RFC3339),
	}

	if c.LastEvent != nil {
		s := c.LastEvent.Format(time.RFC3339)
		ac.LastEvent = &s
	}
	if c.LastThreadAt != nil {
		s := c.LastThreadAt.Format(time.RFC3339)
		ac.LastThreadAt = &s
	}
	if c.Outcome != nil {
		s := c.Outcome.String()
		ac.Outcome = &s
	}

	return ac
}

func toAppContexts(cs []contextbus.Context) []Context {
	contexts := make([]Context, len(cs))
	for i, c := range cs {
		contexts[i] = toAppContext(c)
	}
	return contexts
}

func toBusNewContext(nc NewContext) contextbus.NewContext {
	return contextbus.NewContext{
		Title:       nc.Title,
		Description: nc.Description,
	}
}

func toBusUpdateContext(uc UpdateContext) (contextbus.UpdateContext, error) {
	var buc contextbus.UpdateContext

	buc.Title = uc.Title
	buc.Description = uc.Description
	buc.Summary = uc.Summary

	if uc.Status != nil {
		s, err := contextbus.Parse(*uc.Status)
		if err != nil {
			return contextbus.UpdateContext{}, fmt.Errorf("status: %w", err)
		}
		buc.Status = &s
	}

	if uc.DebriefStatus != nil {
		ds, err := debriefstatus.Parse(*uc.DebriefStatus)
		if err != nil {
			return contextbus.UpdateContext{}, fmt.Errorf("debriefStatus: %w", err)
		}
		buc.DebriefStatus = &ds
	}

	if uc.Outcome != nil {
		o, err := contextoutcome.Parse(*uc.Outcome)
		if err != nil {
			return contextbus.UpdateContext{}, fmt.Errorf("outcome: %w", err)
		}
		buc.Outcome = &o
	}

	return buc, nil
}

func toAppEvent(e contextbus.Event) Event {
	ae := Event{
		ID:        e.ID.String(),
		ContextID: e.ContextID.String(),
		Kind:      e.Kind,
		Content:   e.Content,
		CreatedAt: e.CreatedAt.Format(time.RFC3339),
	}

	if e.Metadata != nil {
		ae.Metadata = *e.Metadata
	}

	if e.SourceID != nil {
		s := e.SourceID.String()
		ae.SourceID = &s
	}

	return ae
}

func toAppEvents(es []contextbus.Event) []Event {
	events := make([]Event, len(es))
	for i, e := range es {
		events[i] = toAppEvent(e)
	}
	return events
}

func toBusNewEvent(ne NewEvent, contextID uuid.UUID) (contextbus.NewEvent, error) {
	bne := contextbus.NewEvent{
		ContextID: contextID,
		Kind:      ne.Kind,
		Content:   ne.Content,
		Metadata:  (*json.RawMessage)(&ne.Metadata),
	}

	if ne.SourceID != nil {
		id, err := uuid.Parse(*ne.SourceID)
		if err != nil {
			return contextbus.NewEvent{}, fmt.Errorf("sourceId: %w", err)
		}
		bne.SourceID = &id
	}

	return bne, nil
}
