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

	// QueryOverlappingContexts returns pairs of active contexts that share 2+
	// tags. Keyword-based heuristic — true similarity requires embeddings.
	QueryOverlappingContexts(ctx context.Context) ([]OverlapPair, error)
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

	if err := b.CheckOverlaps(ctx); err != nil {
		b.log.Error(ctx, "inactivity", "msg", "overlap check failed", "error", err)
	}

	b.log.Info(ctx, "inactivity", "msg", "check complete", "stale_tasks", len(staleTasks), "stale_contexts", len(staleContexts))

	return nil
}

// CheckOverlaps scans for active contexts that share 2+ tags and creates
// overlapping_contexts clarification cards.
func (b *Business) CheckOverlaps(ctx context.Context) error {
	pairs, err := b.storer.QueryOverlappingContexts(ctx)
	if err != nil {
		return fmt.Errorf("query overlapping contexts: %w", err)
	}

	for _, pair := range pairs {
		if err := b.createOverlapPrompt(ctx, pair); err != nil {
			b.log.Error(ctx, "inactivity", "msg", "failed to create overlap prompt", "error", err,
				"context_1", pair.ContextID1, "context_2", pair.ContextID2)
		}
	}

	b.log.Info(ctx, "inactivity", "msg", "overlap check complete", "pairs_found", len(pairs))
	return nil
}

func (b *Business) createOverlapPrompt(ctx context.Context, pair OverlapPair) error {
	kind := clarificationkind.OverlappingContexts
	pending := clarificationstatus.Pending
	subjectType := "context"

	existing, err := b.clarificationBus.Count(ctx, clarificationbus.QueryFilter{
		Kind:        &kind,
		Status:      &pending,
		SubjectType: &subjectType,
		SubjectID:   &pair.ContextID1,
	})
	if err != nil {
		return fmt.Errorf("check existing: %w", err)
	}
	if existing > 0 {
		return nil
	}

	optionsJSON, _ := json.Marshal([]map[string]string{
		{"label": "Keep separate", "action": "keep"},
		{"label": "Merge them", "action": "merge"},
		{"label": "Dismiss", "action": "dismiss"},
	})

	question := fmt.Sprintf("'%s' and '%s' share %d tags — are these overlapping? Should they be merged?",
		pair.Title1, pair.Title2, pair.SharedTags)

	if _, err := b.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{
		Kind:          clarificationkind.OverlappingContexts,
		SubjectType:   "context",
		SubjectID:     pair.ContextID1,
		Question:      question,
		AnswerOptions: json.RawMessage(optionsJSON),
	}); err != nil {
		return fmt.Errorf("create clarification: %w", err)
	}

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
