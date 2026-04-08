package contextdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/types/contextkind"
	"github.com/casebrophy/planner/business/types/contextoutcome"
	"github.com/casebrophy/planner/business/types/debriefstatus"
)

type contextDB struct {
	ID              uuid.UUID  `db:"context_id"`
	Title           string     `db:"title"`
	Description     string     `db:"description"`
	Kind            string     `db:"kind"`
	Status          string     `db:"status"`
	Summary         string     `db:"summary"`
	LastEvent       *time.Time `db:"last_event"`
	LastThreadAt    *time.Time `db:"last_thread_at"`
	DebriefStatus   string     `db:"debrief_status"`
	Outcome         *string    `db:"outcome"`
	ParentContextID *uuid.UUID `db:"parent_context_id"`
	CreatedAt       time.Time  `db:"created_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

func toDBContext(c contextbus.Context) contextDB {
	db := contextDB{
		ID:              c.ID,
		Title:           c.Title,
		Description:     c.Description,
		Kind:            c.Kind.String(),
		Status:          c.Status.String(),
		Summary:         c.Summary,
		LastEvent:       c.LastEvent,
		LastThreadAt:    c.LastThreadAt,
		DebriefStatus:   c.DebriefStatus.String(),
		ParentContextID: c.ParentContextID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if c.Outcome != nil {
		s := c.Outcome.String()
		db.Outcome = &s
	}
	return db
}

func toBusContext(c contextDB) contextbus.Context {
	bc := contextbus.Context{
		ID:              c.ID,
		Title:           c.Title,
		Description:     c.Description,
		Kind:            contextkind.MustParse(c.Kind),
		Status:          contextbus.MustParse(c.Status),
		Summary:         c.Summary,
		LastEvent:       c.LastEvent,
		LastThreadAt:    c.LastThreadAt,
		DebriefStatus:   debriefstatus.MustParse(c.DebriefStatus),
		ParentContextID: c.ParentContextID,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
	if c.Outcome != nil {
		o := contextoutcome.MustParse(*c.Outcome)
		bc.Outcome = &o
	}
	return bc
}

func toBusContexts(cs []contextDB) []contextbus.Context {
	result := make([]contextbus.Context, len(cs))
	for i, c := range cs {
		result[i] = toBusContext(c)
	}
	return result
}

