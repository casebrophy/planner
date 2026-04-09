package classificationcorrectionbus

import (
	"time"

	"github.com/google/uuid"
)

type Correction struct {
	ID            uuid.UUID
	ClauseText    string
	PredictedType string
	Confidence    float64
	ActualType    string
	Source        string // "clarification_answered" | "correction_applied"
	CreatedAt     time.Time
}

type NewCorrection struct {
	ClauseText    string
	PredictedType string
	Confidence    float64
	ActualType    string
	Source        string
}
