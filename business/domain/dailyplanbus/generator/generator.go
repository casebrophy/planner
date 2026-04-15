package generator

import (
	"context"
	"fmt"
	"time"

	"github.com/casebrophy/planner/foundation/claudecli"
)

// TaskRef is a lightweight task reference for the AI prompt.
type TaskRef struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Priority    string  `json:"priority"`
	Energy      string  `json:"energy"`
	DurationMin *int    `json:"duration_min,omitempty"`
	DueDate     *string `json:"due_date,omitempty"`
	Context     *string `json:"context,omitempty"`
	Status      string  `json:"status"`
}

// EventRef is a lightweight event reference for the AI prompt.
type EventRef struct {
	ID       string  `json:"id"`
	Title    string  `json:"title"`
	StartsAt string  `json:"starts_at"`
	EndsAt   string  `json:"ends_at"`
	Location *string `json:"location,omitempty"`
	AllDay   bool    `json:"all_day"`
}

// CarryoverItem is a task from yesterday's plan that wasn't completed.
type CarryoverItem struct {
	TaskID string `json:"task_id"`
	Title  string `json:"title"`
	Reason string `json:"reason"` // why it wasn't completed (dismissed reason or "not completed")
}

// PlanOutput is what Claude returns.
type PlanOutput struct {
	Groups []PlanGroup `json:"groups"`
}

// PlanGroup is a named group of tasks in the plan.
type PlanGroup struct {
	Name   string     `json:"name"`
	Reason string     `json:"reason"`
	Items  []PlanItem `json:"items"`
}

// PlanItem is a single task in the plan.
type PlanItem struct {
	TaskID         string `json:"task_id"`
	AIDurationMin  int    `json:"ai_duration_min"`
	PriorityReason string `json:"priority_reason"`
}

// Generator creates daily plans using Claude.
type Generator struct {
	client *claudecli.Client
}

// NewGenerator creates a new plan generator.
func NewGenerator(client *claudecli.Client) *Generator {
	return &Generator{client: client}
}

// Generate creates a daily plan by analyzing tasks, events, and carryover items.
// It runs implication reasoning before calling the LLM so Claude can schedule
// prep tasks before their associated events.
func (g *Generator) Generate(ctx context.Context, tasks []TaskRef, events []EventRef, carryover []CarryoverItem, tzName string) (PlanOutput, []ImplicationResult, string, error) {
	implications := ReasonImplications(tasks, events, time.Now())
	prompt := buildPlanPrompt(tasks, events, carryover, implications, tzName)

	var output PlanOutput
	shouldEscalate := func() bool {
		return len(output.Groups) == 0
	}

	if err := g.client.RunJSON(ctx, prompt, planSchema, &output, shouldEscalate, claudecli.RunOptions{Direct: true}); err != nil {
		return PlanOutput{}, nil, "", fmt.Errorf("generate plan: %w", err)
	}

	return output, implications, g.client.LastModel(), nil
}

const planSchema = `{
    "type": "object",
    "properties": {
        "groups": {
            "type": "array",
            "items": {
                "type": "object",
                "properties": {
                    "name": {"type": "string"},
                    "reason": {"type": "string"},
                    "items": {
                        "type": "array",
                        "items": {
                            "type": "object",
                            "properties": {
                                "task_id": {"type": "string"},
                                "ai_duration_min": {"type": "integer"},
                                "priority_reason": {"type": "string"}
                            },
                            "required": ["task_id", "ai_duration_min", "priority_reason"]
                        }
                    }
                },
                "required": ["name", "reason", "items"]
            }
        }
    },
    "required": ["groups"]
}`
