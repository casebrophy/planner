package debriefbus

import (
	"github.com/google/uuid"
)

// CompletedTask holds the data needed to evaluate debrief triggers for a task.
type CompletedTask struct {
	ID          uuid.UUID
	Title       string
	DurationMin *int
	CreatedAt   int64 // unix timestamp
	CompletedAt int64 // unix timestamp
}

// ClosedContext holds the data needed to generate context debrief cards.
type ClosedContext struct {
	ID    uuid.UUID
	Title string
}
