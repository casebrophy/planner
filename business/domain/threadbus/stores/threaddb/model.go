package threaddb

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

type threadEntryDB struct {
	ID             uuid.UUID        `db:"entry_id"`
	SubjectType    string           `db:"subject_type"`
	SubjectID      uuid.UUID        `db:"subject_id"`
	Kind           string           `db:"kind"`
	Content        string           `db:"content"`
	Metadata       *json.RawMessage `db:"metadata"`
	Source         string           `db:"source"`
	SourceID       *uuid.UUID       `db:"source_id"`
	Sentiment      *string          `db:"sentiment"`
	RequiresAction bool             `db:"requires_action"`
	CreatedAt      time.Time        `db:"created_at"`
}

func toDBThreadEntry(e threadbus.ThreadEntry) threadEntryDB {
	return threadEntryDB{
		ID:             e.ID,
		SubjectType:    e.SubjectType,
		SubjectID:      e.SubjectID,
		Kind:           e.Kind.String(),
		Content:        e.Content,
		Metadata:       e.Metadata,
		Source:         e.Source.String(),
		SourceID:       e.SourceID,
		Sentiment:      e.Sentiment,
		RequiresAction: e.RequiresAction,
		CreatedAt:      e.CreatedAt,
	}
}

func toBusThreadEntry(e threadEntryDB) threadbus.ThreadEntry {
	return threadbus.ThreadEntry{
		ID:             e.ID,
		SubjectType:    e.SubjectType,
		SubjectID:      e.SubjectID,
		Kind:           threadentrykind.MustParse(e.Kind),
		Content:        e.Content,
		Metadata:       e.Metadata,
		Source:         threadsource.MustParse(e.Source),
		SourceID:       e.SourceID,
		Sentiment:      e.Sentiment,
		RequiresAction: e.RequiresAction,
		CreatedAt:      e.CreatedAt,
	}
}

func toBusThreadEntries(es []threadEntryDB) []threadbus.ThreadEntry {
	result := make([]threadbus.ThreadEntry, len(es))
	for i, e := range es {
		result[i] = toBusThreadEntry(e)
	}
	return result
}
