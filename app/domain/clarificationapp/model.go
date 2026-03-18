package clarificationapp

import (
	"encoding/json"
	"time"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
)

type ClarificationItem struct {
	ID            string          `json:"id"`
	Kind          string          `json:"kind"`
	Status        string          `json:"status"`
	SubjectType   string          `json:"subjectType"`
	SubjectID     string          `json:"subjectId"`
	Question      string          `json:"question"`
	ClaudeGuess   json.RawMessage `json:"claudeGuess,omitempty"`
	Reasoning     *string         `json:"reasoning,omitempty"`
	AnswerOptions json.RawMessage `json:"answerOptions"`
	Answer        json.RawMessage `json:"answer,omitempty"`
	PriorityScore float32         `json:"priorityScore"`
	SnoozedUntil  *string         `json:"snoozedUntil,omitempty"`
	CreatedAt     string          `json:"createdAt"`
	ResolvedAt    *string         `json:"resolvedAt,omitempty"`
}

func (c ClarificationItem) Encode() ([]byte, string, error) {
	data, err := json.Marshal(c)
	return data, "application/json", err
}

type ResolveInput struct {
	Answer json.RawMessage `json:"answer"`
}

type SnoozeInput struct {
	Hours int `json:"hours"`
}

type CountResponse struct {
	Count int `json:"count"`
}

func (c CountResponse) Encode() ([]byte, string, error) {
	data, err := json.Marshal(c)
	return data, "application/json", err
}

func toAppClarification(c clarificationbus.ClarificationItem) ClarificationItem {
	ac := ClarificationItem{
		ID:            c.ID.String(),
		Kind:          c.Kind.String(),
		Status:        c.Status.String(),
		SubjectType:   c.SubjectType,
		SubjectID:     c.SubjectID.String(),
		Question:      c.Question,
		Reasoning:     c.Reasoning,
		AnswerOptions: c.AnswerOptions,
		PriorityScore: c.PriorityScore,
		CreatedAt:     c.CreatedAt.Format(time.RFC3339),
	}

	if c.ClaudeGuess != nil {
		ac.ClaudeGuess = *c.ClaudeGuess
	}

	if c.Answer != nil {
		ac.Answer = *c.Answer
	}

	if c.SnoozedUntil != nil {
		s := c.SnoozedUntil.Format(time.RFC3339)
		ac.SnoozedUntil = &s
	}

	if c.ResolvedAt != nil {
		s := c.ResolvedAt.Format(time.RFC3339)
		ac.ResolvedAt = &s
	}

	return ac
}

func toAppClarifications(cs []clarificationbus.ClarificationItem) []ClarificationItem {
	result := make([]ClarificationItem, len(cs))
	for i, c := range cs {
		result[i] = toAppClarification(c)
	}
	return result
}
