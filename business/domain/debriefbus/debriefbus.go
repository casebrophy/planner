package debriefbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/business/types/threadentrykind"
	"github.com/casebrophy/planner/foundation/logger"
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

// OnTaskCompleted evaluates whether a debrief card should be generated for a
// completed task. A card is generated when:
//   - Actual duration > 2x estimated duration, OR
//   - Thread contains blocker entries
func (b *Business) OnTaskCompleted(ctx context.Context, ct CompletedTask) error {
	kind := clarificationkind.TaskDebrief
	pending := clarificationstatus.Pending
	subjectType := "task"
	existing, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &ct.ID,
	})
	if err != nil {
		return fmt.Errorf("check existing debrief: %w", err)
	}
	if existing > 0 {
		return nil
	}

	hasBlockers, err := b.hasBlockerEntries(ctx, "task", ct.ID)
	if err != nil {
		return fmt.Errorf("check blockers: %w", err)
	}

	durationOverrun := false
	if ct.DurationMin != nil && *ct.DurationMin > 0 {
		actualMinutes := float64(ct.CompletedAt-ct.CreatedAt) / 60
		estimatedMinutes := float64(*ct.DurationMin)
		if actualMinutes > estimatedMinutes*2 {
			durationOverrun = true
		}
	}

	if !hasBlockers && !durationOverrun {
		return nil
	}

	question := fmt.Sprintf("Task '%s' is done.", ct.Title)
	if durationOverrun {
		question += " It took significantly longer than estimated. What caused the overrun?"
	} else if hasBlockers {
		question += " It had blockers along the way. What finally unblocked it?"
	}

	optionsJSON, err := json.Marshal([]map[string]string{
		{"label": "Add a lesson", "action": "lesson"},
		{"label": "Nothing notable", "action": "skip"},
		{"label": "Snooze", "action": "snooze"},
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

func (b *Business) hasBlockerEntries(ctx context.Context, subjectType string, subjectID uuid.UUID) (bool, error) {
	blockerKind := threadentrykind.Blocker
	filter := threadbus.QueryFilter{
		SubjectType: &subjectType,
		SubjectID:   &subjectID,
		Kind:        &blockerKind,
	}

	count, err := b.threadBus.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("count blockers: %w", err)
	}

	return count > 0, nil
}
