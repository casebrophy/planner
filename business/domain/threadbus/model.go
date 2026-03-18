package threadbus

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/business/types/threadsource"
)

type ThreadEntry struct {
	ID             uuid.UUID
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
	CreatedAt      time.Time
}

type NewThreadEntry struct {
	SubjectType    string
	SubjectID      uuid.UUID
	Kind           threadentrykind.Kind
	Content        string
	Metadata       *json.RawMessage
	Source         threadsource.Source
	SourceID       *uuid.UUID
	Sentiment      *string
	RequiresAction bool
}
