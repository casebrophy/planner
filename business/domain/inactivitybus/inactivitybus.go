package inactivitybus

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
	"github.com/casebrophy/planner/foundation/logger"
)

// Storer defines the data access interface for inactivity detection.
type Storer interface {
	// QueryStaleTasks returns tasks that have exceeded their priority-based
	// inactivity threshold. Thresholds: urgent=1d, high=2d, medium=5d, low=14d.
	QueryStaleTasks(ctx context.Context) ([]StaleItem, error)

	// QueryStaleContexts returns active contexts that have exceeded their
	// expected_update_days or 7d default inactivity threshold.
	QueryStaleContexts(ctx context.Context) ([]StaleItem, error)
}

// Business manages inactivity detection and clarification generation.
type Business struct {
	log              *logger.Logger
	storer           Storer
	clarificationBus *clarificationbus.Business
}

// NewBusiness creates a new inactivity detection business layer.
func NewBusiness(log *logger.Logger, storer Storer, clarificationBus *clarificationbus.Business) *Business {
	return &Business{
		log:              log,
		storer:           storer,
		clarificationBus: clarificationBus,
	}
}

// CheckAll scans for stale tasks and contexts and creates inactivity_prompt
// clarification items for each. It skips subjects that already have a pending
// inactivity_prompt clarification.
func (b *Business) CheckAll(ctx context.Context) error {
	staleTasks, err := b.storer.QueryStaleTasks(ctx)
	if err != nil {
		return fmt.Errorf("query stale tasks: %w", err)
	}

	for _, item := range staleTasks {
		if err := b.createInactivityPrompt(ctx, item); err != nil {
			b.log.Error(ctx, "inactivity", "msg", "failed to create task inactivity prompt", "error", err, "subject_id", item.SubjectID)
		}
	}

	staleContexts, err := b.storer.QueryStaleContexts(ctx)
	if err != nil {
		return fmt.Errorf("query stale contexts: %w", err)
	}

	for _, item := range staleContexts {
		if err := b.createInactivityPrompt(ctx, item); err != nil {
			b.log.Error(ctx, "inactivity", "msg", "failed to create context inactivity prompt", "error", err, "subject_id", item.SubjectID)
		}
	}

	b.log.Info(ctx, "inactivity", "msg", "check complete", "stale_tasks", len(staleTasks), "stale_contexts", len(staleContexts))

	return nil
}

func (b *Business) createInactivityPrompt(ctx context.Context, item StaleItem) error {
	// Check for existing pending inactivity_prompt for this subject
	kind := clarificationkind.InactivityPrompt
	pending := clarificationstatus.Pending
	existing, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &item.SubjectType,
		SubjectID:   &item.SubjectID,
	})
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing > 0 {
		return nil // already has a pending prompt
	}

	optionsJSON, _ := json.Marshal(map[string]any{
		"type":           "inactivity_prompt",
		"priority":       item.Priority,
		"last_updated":   item.LastUpdated,
		"threshold_days": item.ThresholdDays,
	})

	question := fmt.Sprintf("No updates on '%s' (%s) for %.0f+ days. What's the status?", item.Title, item.SubjectType, item.ThresholdDays)

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.InactivityPrompt,
		SubjectType:   item.SubjectType,
		SubjectID:     item.SubjectID,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
	}); err != nil {
		return fmt.Errorf("create clarification: %w", err)
	}

	return nil
}
