package entitylinkbus

import (
	"time"

	"github.com/google/uuid"
)

// EntityLink is a directional semantic link between two entities.
// At query time, always fetch both directions (QueryBySource + QueryByTarget).
type EntityLink struct {
	ID         uuid.UUID
	SourceType string // "task" | "note" | "event"
	SourceID   uuid.UUID
	TargetType string // "task" | "note" | "event"
	TargetID   uuid.UUID
	Confidence float64 // 1.0 for manual; 0.0–1.0 for AI-suggested
	Kind       string  // "manual" | "ai_suggested"
	CreatedAt  time.Time
}

// NewEntityLink is the input for creating an entity link.
type NewEntityLink struct {
	SourceType string
	SourceID   uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Confidence float64
	Kind       string
}
