package inactivitybus

import (
	"time"

	"github.com/google/uuid"
)

// StaleItem represents a task or context that has gone stale based on
// priority-driven thresholds.
type StaleItem struct {
	SubjectType string
	SubjectID   uuid.UUID
	Title       string
	Priority    string
	LastUpdated time.Time
	ThresholdDays float64
}
