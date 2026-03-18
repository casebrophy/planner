package clarificationbus

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

type ClarificationItem struct {
	ID            uuid.UUID
	Kind          clarificationkind.Kind
	Status        clarificationstatus.Status
	SubjectType   string
	SubjectID     uuid.UUID
	Question      string
	ClaudeGuess   *json.RawMessage
	Reasoning     *string
	AnswerOptions json.RawMessage
	Answer        *json.RawMessage
	PriorityScore float32
	SnoozedUntil  *time.Time
	CreatedAt     time.Time
	ResolvedAt    *time.Time
}

type NewClarificationItem struct {
	Kind          clarificationkind.Kind
	SubjectType   string
	SubjectID     uuid.UUID
	Question      string
	ClaudeGuess   *json.RawMessage
	Reasoning     *string
	AnswerOptions json.RawMessage
	PriorityScore float32
	SnoozedUntil  *time.Time
}

type ResolveClarificationItem struct {
	Answer json.RawMessage
}
