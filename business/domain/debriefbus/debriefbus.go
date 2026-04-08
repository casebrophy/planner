package debriefbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/google/uuid"
)

// Business manages debrief card generation on task/context completion.
type Business struct {
	log              *logger.Logger
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
}

// NewBusiness creates a new debrief business layer.
func NewBusiness(log *logger.Logger, clarificationBus *clarificationbus.Business, threadBus *threadbus.Business) *Business {
	return &Business{
		log:              log,
		clarificationBus: clarificationBus,
		threadBus:        threadBus,
	}
}

// OnTaskCompleted generates a task debrief card for every completed task.
// The question adapts based on whether the task had a duration estimate and
// whether it overran.
func (b *Business) OnTaskCompleted(ctx context.Context, ct CompletedTask) error {
	kind := clarificationkind.TaskDebrief
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "task"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed debrief: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	question := fmt.Sprintf("You completed '%s'. How important was this?", ct.Title)
	if ct.DurationMin != nil && *ct.DurationMin > 0 {
		actualMinutes := float64(ct.CompletedAt-ct.CreatedAt) / 60
		estimatedMinutes := float64(*ct.DurationMin)
		if actualMinutes > estimatedMinutes*2 {
			question = fmt.Sprintf("You completed '%s' — it took much longer than the %d min estimate. How important was this?", ct.Title, *ct.DurationMin)
		}
	}

	optionsJSON, err := json.Marshal([]map[string]string{
		{"label": "High impact", "value": "high"},
		{"label": "Medium impact", "value": "medium"},
		{"label": "Low impact", "value": "low"},
		{"label": "Not worth it", "value": "waste"},
		{"label": "Skip", "value": "skip"},
	})
	if err != nil {
		return fmt.Errorf("marshal options: %w", err)
	}

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.TaskDebrief,
		SubjectType:   "task",
		SubjectID:     ct.ID,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
		PriorityScore: 0.9,
	}); err != nil {
		return fmt.Errorf("create task debrief: %w", err)
	}

	return nil
}

// OnContextClosed generates a 3-card closing review sequence for a closed context.
// Cards are created with snoozed_until = now + 24h so they surface after a cooling-off period.
func (b *Business) OnContextClosed(ctx context.Context, cc ClosedContext) error {
	kind := clarificationkind.ContextDebrief
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "context"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &cc.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &cc.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed debrief: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	snoozedUntil := time.Now().Add(24 * time.Hour)

	// Card 1: Outcome
	outcomeOptions, err := json.Marshal([]map[string]string{
		{"label": "Went well", "action": "went_well"},
		{"label": "Mixed results", "action": "mixed"},
		{"label": "Difficult", "action": "difficult"},
		{"label": "Skip debrief", "action": "skip"},
	})
	if err != nil {
		return fmt.Errorf("marshal outcome options: %w", err)
	}
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("Context '%s' is closed. How did it go overall?", cc.Title),
		AnswerOptions: json.RawMessage(outcomeOptions),
		PriorityScore: 0.8,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create outcome card: %w", err)
	}

	// Card 2: Biggest challenge
	challengeOptions, err := json.Marshal([]map[string]string{
		{"label": "Timeline pressure", "action": "timeline"},
		{"label": "Unclear requirements", "action": "requirements"},
		{"label": "External dependencies", "action": "dependencies"},
		{"label": "No major challenges", "action": "none"},
	})
	if err != nil {
		return fmt.Errorf("marshal challenge options: %w", err)
	}
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("What was the biggest challenge with '%s'?", cc.Title),
		AnswerOptions: json.RawMessage(challengeOptions),
		PriorityScore: 0.7,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create challenge card: %w", err)
	}

	// Card 3: Lesson
	lessonOptions, err := json.Marshal([]map[string]string{
		{"label": "Add a lesson", "action": "lesson"},
		{"label": "Nothing to add", "action": "skip"},
	})
	if err != nil {
		return fmt.Errorf("marshal lesson options: %w", err)
	}
	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.ContextDebrief,
		SubjectType:   "context",
		SubjectID:     cc.ID,
		Question:      fmt.Sprintf("Any lessons or insights from '%s' worth remembering?", cc.Title),
		AnswerOptions: json.RawMessage(lessonOptions),
		PriorityScore: 0.6,
		SnoozedUntil:  &snoozedUntil,
	}); err != nil {
		return fmt.Errorf("create lesson card: %w", err)
	}

	return nil
}

// GenerateWeeklyReview creates a weekly impact review card from the given
// completed tasks. The caller (scheduler in main.go) queries and provides
// the task list. weekID is an ISO week string like "2026-W15" used for dedup.
func (b *Business) GenerateWeeklyReview(ctx context.Context, weekID string, tasks []CompletedTask) error {
	if len(tasks) == 0 {
		return nil
	}

	// Deterministic UUID from week string for dedup
	subjectID := uuid.NewSHA1(uuid.NameSpaceURL, []byte("planner:weekly-review:"+weekID))

	kind := clarificationkind.WeeklyReview
	pending := clarificationstatus.Pending
	snoozed := clarificationstatus.Snoozed
	subjectType := "week"

	existingPending, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return fmt.Errorf("check existing weekly review: %w", err)
	}

	existingSnoozed, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &snoozed,
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
	})
	if err != nil {
		return fmt.Errorf("check existing snoozed weekly review: %w", err)
	}

	if existingPending > 0 || existingSnoozed > 0 {
		return nil
	}

	// Build task list for answer options
	reviewTasks := make([]WeeklyReviewTask, len(tasks))
	for i, t := range tasks {
		reviewTasks[i] = WeeklyReviewTask{
			ID:    t.ID,
			Title: t.Title,
		}
	}

	optionsJSON, err := json.Marshal(map[string]any{
		"tasks": reviewTasks,
	})
	if err != nil {
		return fmt.Errorf("marshal weekly review options: %w", err)
	}

	question := fmt.Sprintf("You completed %d tasks this week. Which had the most impact?", len(tasks))

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.WeeklyReview,
		SubjectType:   "week",
		SubjectID:     subjectID,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
		PriorityScore: 0.8,
	}); err != nil {
		return fmt.Errorf("create weekly review: %w", err)
	}

	return nil
}
