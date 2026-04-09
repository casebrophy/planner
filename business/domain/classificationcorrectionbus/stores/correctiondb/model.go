package correctiondb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/classificationcorrectionbus"
)

type correctionDB struct {
	ID            uuid.UUID `db:"correction_id"`
	ClauseText    string    `db:"clause_text"`
	PredictedType string    `db:"predicted_type"`
	Confidence    float64   `db:"confidence"`
	ActualType    string    `db:"actual_type"`
	Source        string    `db:"source"`
	CreatedAt     time.Time `db:"created_at"`
}

func toDBCorrection(c classificationcorrectionbus.Correction) correctionDB {
	return correctionDB{
		ID:            c.ID,
		ClauseText:    c.ClauseText,
		PredictedType: c.PredictedType,
		Confidence:    c.Confidence,
		ActualType:    c.ActualType,
		Source:        c.Source,
		CreatedAt:     c.CreatedAt,
	}
}

func toBusCorrection(c correctionDB) classificationcorrectionbus.Correction {
	return classificationcorrectionbus.Correction{
		ID:            c.ID,
		ClauseText:    c.ClauseText,
		PredictedType: c.PredictedType,
		Confidence:    c.Confidence,
		ActualType:    c.ActualType,
		Source:        c.Source,
		CreatedAt:     c.CreatedAt,
	}
}

func toBusCorrections(cs []correctionDB) []classificationcorrectionbus.Correction {
	result := make([]classificationcorrectionbus.Correction, len(cs))
	for i, c := range cs {
		result[i] = toBusCorrection(c)
	}
	return result
}
