package threadapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

type ThreadEntry struct {
	ID             string          `json:"id"`
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId,omitempty"`
	Sentiment      *string         `json:"sentiment,omitempty"`
	RequiresAction bool            `json:"requiresAction"`
	CreatedAt      string          `json:"createdAt"`
}

func (e ThreadEntry) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

type NewThreadEntry struct {
	SubjectType    string          `json:"subjectType"`
	SubjectID      string          `json:"subjectId"`
	Kind           string          `json:"kind"`
	Content        string          `json:"content"`
	Metadata       json.RawMessage `json:"metadata,omitempty"`
	Source         string          `json:"source"`
	SourceID       *string         `json:"sourceId"`
	Sentiment      *string         `json:"sentiment"`
	RequiresAction bool            `json:"requiresAction"`
}

func toAppThreadEntry(e threadbus.ThreadEntry) ThreadEntry {
	ae := ThreadEntry{
		ID:             e.ID.String(),
		SubjectType:    e.SubjectType,
		SubjectID:      e.SubjectID.String(),
		Kind:           e.Kind.String(),
		Content:        e.Content,
		Source:         e.Source.String(),
		Sentiment:      e.Sentiment,
		RequiresAction: e.RequiresAction,
		CreatedAt:      e.CreatedAt.Format(time.RFC3339),
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

func toAppThreadEntries(es []threadbus.ThreadEntry) []ThreadEntry {
	entries := make([]ThreadEntry, len(es))
	for i, e := range es {
		entries[i] = toAppThreadEntry(e)
	}
	return entries
}

func toBusNewThreadEntry(ne NewThreadEntry) (threadbus.NewThreadEntry, error) {
	subjectID, err := uuid.Parse(ne.SubjectID)
	if err != nil {
		return threadbus.NewThreadEntry{}, fmt.Errorf("subjectId: %w", err)
	}

	kind, err := threadentrykind.Parse(ne.Kind)
	if err != nil {
		return threadbus.NewThreadEntry{}, fmt.Errorf("kind: %w", err)
	}

	source := threadsource.User
	if ne.Source != "" {
		source, err = threadsource.Parse(ne.Source)
		if err != nil {
			return threadbus.NewThreadEntry{}, fmt.Errorf("source: %w", err)
		}
	}

	bne := threadbus.NewThreadEntry{
		SubjectType:    ne.SubjectType,
		SubjectID:      subjectID,
		Kind:           kind,
		Content:        ne.Content,
		Source:         source,
		Sentiment:      ne.Sentiment,
		RequiresAction: ne.RequiresAction,
	}

	if len(ne.Metadata) > 0 {
		raw := json.RawMessage(ne.Metadata)
		bne.Metadata = &raw
	}

	if ne.SourceID != nil {
		id, err := uuid.Parse(*ne.SourceID)
		if err != nil {
			return threadbus.NewThreadEntry{}, fmt.Errorf("sourceId: %w", err)
		}
		bne.SourceID = &id
	}

	return bne, nil
}
