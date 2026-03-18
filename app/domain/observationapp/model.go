package observationapp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/types/observationkind"
)

type Observation struct {
	ID          string          `json:"id"`
	SubjectType string          `json:"subjectType"`
	SubjectID   string          `json:"subjectId"`
	Kind        string          `json:"kind"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source"`
	Confidence  float32         `json:"confidence"`
	Weight      float32         `json:"weight"`
	CreatedAt   string          `json:"createdAt"`
}

func (o Observation) Encode() ([]byte, string, error) {
	data, err := json.Marshal(o)
	return data, "application/json", err
}

type NewObservation struct {
	SubjectType string          `json:"subjectType"`
	SubjectID   string          `json:"subjectId"`
	Kind        string          `json:"kind"`
	Data        json.RawMessage `json:"data"`
	Source      string          `json:"source"`
	Confidence  *float32        `json:"confidence"`
	Weight      *float32        `json:"weight"`
}

func toAppObservation(o observationbus.Observation) Observation {
	return Observation{
		ID:          o.ID.String(),
		SubjectType: o.SubjectType,
		SubjectID:   o.SubjectID.String(),
		Kind:        o.Kind.String(),
		Data:        o.Data,
		Source:      o.Source,
		Confidence:  o.Confidence,
		Weight:      o.Weight,
		CreatedAt:   o.CreatedAt.Format(time.RFC3339),
	}
}

func toAppObservations(os []observationbus.Observation) []Observation {
	result := make([]Observation, len(os))
	for i, o := range os {
		result[i] = toAppObservation(o)
	}
	return result
}

func toBusNewObservation(no NewObservation) (observationbus.NewObservation, error) {
	subjectID, err := uuid.Parse(no.SubjectID)
	if err != nil {
		return observationbus.NewObservation{}, fmt.Errorf("subjectId: %w", err)
	}

	kind, err := observationkind.Parse(no.Kind)
	if err != nil {
		return observationbus.NewObservation{}, fmt.Errorf("kind: %w", err)
	}

	source := "user"
	if no.Source != "" {
		source = no.Source
	}

	confidence := float32(1.0)
	if no.Confidence != nil {
		confidence = *no.Confidence
	}

	weight := float32(1.0)
	if no.Weight != nil {
		weight = *no.Weight
	}

	return observationbus.NewObservation{
		SubjectType: no.SubjectType,
		SubjectID:   subjectID,
		Kind:        kind,
		Data:        no.Data,
		Source:      source,
		Confidence:  confidence,
		Weight:      weight,
	}, nil
}
