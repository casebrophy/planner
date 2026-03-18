package observationbus

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/observationkind"
)

type Observation struct {
	ID          uuid.UUID
	SubjectType string
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage
	Source      string
	Confidence  float32
	Weight      float32
	CreatedAt   time.Time
}

type NewObservation struct {
	SubjectType string
	SubjectID   uuid.UUID
	Kind        observationkind.Kind
	Data        json.RawMessage
	Source      string
	Confidence  float32
	Weight      float32
}
