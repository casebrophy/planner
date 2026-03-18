package observationdb

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/types/observationkind"
)

type observationDB struct {
	ID          uuid.UUID       `db:"observation_id"`
	SubjectType string          `db:"subject_type"`
	SubjectID   uuid.UUID       `db:"subject_id"`
	Kind        string          `db:"kind"`
	Data        json.RawMessage `db:"data"`
	Source      string          `db:"source"`
	Confidence  float32         `db:"confidence"`
	Weight      float32         `db:"weight"`
	CreatedAt   time.Time       `db:"created_at"`
}

func toDBObservation(o observationbus.Observation) observationDB {
	return observationDB{
		ID:          o.ID,
		SubjectType: o.SubjectType,
		SubjectID:   o.SubjectID,
		Kind:        o.Kind.String(),
		Data:        o.Data,
		Source:      o.Source,
		Confidence:  o.Confidence,
		Weight:      o.Weight,
		CreatedAt:   o.CreatedAt,
	}
}

func toBusObservation(o observationDB) observationbus.Observation {
	return observationbus.Observation{
		ID:          o.ID,
		SubjectType: o.SubjectType,
		SubjectID:   o.SubjectID,
		Kind:        observationkind.MustParse(o.Kind),
		Data:        o.Data,
		Source:      o.Source,
		Confidence:  o.Confidence,
		Weight:      o.Weight,
		CreatedAt:   o.CreatedAt,
	}
}

func toBusObservations(os []observationDB) []observationbus.Observation {
	result := make([]observationbus.Observation, len(os))
	for i, o := range os {
		result[i] = toBusObservation(o)
	}
	return result
}
