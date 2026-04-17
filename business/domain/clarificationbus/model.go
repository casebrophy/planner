package clarificationbus

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

// ErrInvalidClarification is returned by NewClarificationItem.Validate when a
// required field is missing. Callers can use errors.Is to detect it.
var ErrInvalidClarification = errors.New("invalid clarification item")

type ClarificationItem struct {
	ID                 uuid.UUID
	Kind               clarificationkind.Kind
	Status             clarificationstatus.Status
	SubjectType        string
	SubjectID          uuid.UUID
	SubjectDescription string
	GapCategory        string
	Question           string
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
	Kind               clarificationkind.Kind
	SubjectType        string
	SubjectID          uuid.UUID
	SubjectDescription string
	GapCategory        string
	Question           string
	ClaudeGuess   *json.RawMessage
	Reasoning     *string
	AnswerOptions json.RawMessage
	PriorityScore float32
	SnoozedUntil  *time.Time
}

type ResolveClarificationItem struct {
	Answer json.RawMessage
}

// Validate enforces required fields so clarification cards are never persisted
// without the context needed to render them to the user.
func (nc NewClarificationItem) Validate() error {
	var missing []string
	if nc.Kind == (clarificationkind.Kind{}) {
		missing = append(missing, "kind")
	}
	if strings.TrimSpace(nc.SubjectType) == "" {
		missing = append(missing, "subject_type")
	}
	if nc.SubjectID == uuid.Nil {
		missing = append(missing, "subject_id")
	}
	if strings.TrimSpace(nc.SubjectDescription) == "" {
		missing = append(missing, "subject_description")
	}
	if strings.TrimSpace(nc.Question) == "" {
		missing = append(missing, "question")
	}
	if len(nc.AnswerOptions) == 0 {
		missing = append(missing, "answer_options")
	}
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("%w: missing %s", ErrInvalidClarification, strings.Join(missing, ", "))
}
