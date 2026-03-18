package clarificationdb

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

type clarificationDB struct {
	ID            uuid.UUID        `db:"clarification_id"`
	Kind          string           `db:"kind"`
	Status        string           `db:"status"`
	SubjectType   string           `db:"subject_type"`
	SubjectID     uuid.UUID        `db:"subject_id"`
	Question      string           `db:"question"`
	ClaudeGuess   *json.RawMessage `db:"claude_guess"`
	Reasoning     *string          `db:"reasoning"`
	AnswerOptions json.RawMessage  `db:"answer_options"`
	Answer        *json.RawMessage `db:"answer"`
	PriorityScore float32          `db:"priority_score"`
	SnoozedUntil  *time.Time       `db:"snoozed_until"`
	CreatedAt     time.Time        `db:"created_at"`
	ResolvedAt    *time.Time       `db:"resolved_at"`
}

func toDBClarification(c clarificationbus.ClarificationItem) clarificationDB {
	return clarificationDB{
		ID:            c.ID,
		Kind:          c.Kind.String(),
		Status:        c.Status.String(),
		SubjectType:   c.SubjectType,
		SubjectID:     c.SubjectID,
		Question:      c.Question,
		ClaudeGuess:   c.ClaudeGuess,
		Reasoning:     c.Reasoning,
		AnswerOptions: c.AnswerOptions,
		Answer:        c.Answer,
		PriorityScore: c.PriorityScore,
		SnoozedUntil:  c.SnoozedUntil,
		CreatedAt:     c.CreatedAt,
		ResolvedAt:    c.ResolvedAt,
	}
}

func toBusClarification(c clarificationDB) clarificationbus.ClarificationItem {
	return clarificationbus.ClarificationItem{
		ID:            c.ID,
		Kind:          clarificationkind.MustParse(c.Kind),
		Status:        clarificationstatus.MustParse(c.Status),
		SubjectType:   c.SubjectType,
		SubjectID:     c.SubjectID,
		Question:      c.Question,
		ClaudeGuess:   c.ClaudeGuess,
		Reasoning:     c.Reasoning,
		AnswerOptions: c.AnswerOptions,
		Answer:        c.Answer,
		PriorityScore: c.PriorityScore,
		SnoozedUntil:  c.SnoozedUntil,
		CreatedAt:     c.CreatedAt,
		ResolvedAt:    c.ResolvedAt,
	}
}

func toBusClarifications(cs []clarificationDB) []clarificationbus.ClarificationItem {
	result := make([]clarificationbus.ClarificationItem, len(cs))
	for i, c := range cs {
		result[i] = toBusClarification(c)
	}
	return result
}
