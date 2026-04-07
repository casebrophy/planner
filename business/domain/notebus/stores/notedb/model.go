package notedb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/notebus"
)

type noteDB struct {
	ID         uuid.UUID  `db:"note_id"`
	ContextID  *uuid.UUID `db:"context_id"`
	TaskID     *uuid.UUID `db:"task_id"`
	Content    string     `db:"content"`
	Source     string     `db:"source"`
	RawInputID *uuid.UUID `db:"raw_input_id"`
	CreatedAt  time.Time  `db:"created_at"`
	UpdatedAt  time.Time  `db:"updated_at"`
}

func toDBNote(n notebus.Note) noteDB {
	return noteDB{
		ID:         n.ID,
		ContextID:  n.ContextID,
		TaskID:     n.TaskID,
		Content:    n.Content,
		Source:     n.Source,
		RawInputID: n.RawInputID,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
	}
}

func toBusNote(n noteDB) notebus.Note {
	return notebus.Note{
		ID:         n.ID,
		ContextID:  n.ContextID,
		TaskID:     n.TaskID,
		Content:    n.Content,
		Source:     n.Source,
		RawInputID: n.RawInputID,
		CreatedAt:  n.CreatedAt,
		UpdatedAt:  n.UpdatedAt,
	}
}

func toBusNotes(ns []noteDB) []notebus.Note {
	notes := make([]notebus.Note, len(ns))
	for i, n := range ns {
		notes[i] = toBusNote(n)
	}
	return notes
}
