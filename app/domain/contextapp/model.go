package contextapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/types/contextkind"
	"github.com/casebrophy/planner/business/types/contextoutcome"
	"github.com/casebrophy/planner/business/types/debriefstatus"
)

type Context struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Status          string  `json:"status"`
	Kind            string  `json:"kind"`
	Summary         string  `json:"summary"`
	LastEvent       *string `json:"lastEvent,omitempty"`
	LastThreadAt    *string `json:"lastThreadAt,omitempty"`
	DebriefStatus   string  `json:"debriefStatus"`
	Outcome         *string `json:"outcome,omitempty"`
	ParentContextID *string `json:"parentContextId,omitempty"`
	CreatedAt       string  `json:"createdAt"`
	UpdatedAt       string  `json:"updatedAt"`
}

func (c Context) Encode() ([]byte, string, error) {
	data, err := json.Marshal(c)
	return data, "application/json", err
}

type NewContext struct {
	Title           string  `json:"title"`
	Description     string  `json:"description"`
	Kind            string  `json:"kind"`
	ParentContextID *string `json:"parentContextId,omitempty"`
}

type UpdateContext struct {
	Title           *string `json:"title"`
	Description     *string `json:"description"`
	Status          *string `json:"status"`
	Kind            *string `json:"kind"`
	Summary         *string `json:"summary"`
	DebriefStatus   *string `json:"debriefStatus"`
	Outcome         *string `json:"outcome"`
	ParentContextID *string `json:"parentContextId,omitempty"`
}

func toAppContext(c contextbus.Context) Context {
	ac := Context{
		ID:            c.ID.String(),
		Title:         c.Title,
		Description:   c.Description,
		Status:        c.Status.String(),
		Kind:          c.Kind.String(),
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
	if c.ParentContextID != nil {
		s := c.ParentContextID.String()
		ac.ParentContextID = &s
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

func toBusNewContext(nc NewContext) (contextbus.NewContext, error) {
	kind := contextkind.Project
	if nc.Kind != "" {
		var err error
		kind, err = contextkind.Parse(nc.Kind)
		if err != nil {
			return contextbus.NewContext{}, fmt.Errorf("parsing kind: %w", err)
		}
	}

	bnc := contextbus.NewContext{
		Title:       nc.Title,
		Description: nc.Description,
		Kind:        kind,
	}

	if nc.ParentContextID != nil {
		id, err := uuid.Parse(*nc.ParentContextID)
		if err != nil {
			return contextbus.NewContext{}, fmt.Errorf("parentContextId: %w", err)
		}
		bnc.ParentContextID = &id
	}

	return bnc, nil
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

	if uc.Kind != nil {
		kind, err := contextkind.Parse(*uc.Kind)
		if err != nil {
			return contextbus.UpdateContext{}, fmt.Errorf("parsing kind: %w", err)
		}
		buc.Kind = &kind
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

	if uc.ParentContextID != nil {
		id, err := uuid.Parse(*uc.ParentContextID)
		if err != nil {
			return contextbus.UpdateContext{}, fmt.Errorf("parentContextId: %w", err)
		}
		buc.ParentContextID = &id
	}

	return buc, nil
}

