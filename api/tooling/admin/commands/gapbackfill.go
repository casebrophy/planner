package commands

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
)

// GapBackfillCmd implements the gap-backfill admin command.
type GapBackfillCmd struct {
	dryRun     bool
	entityType string
	limit      int
}

// Run executes the gap-backfill command.
func (cmd *GapBackfillCmd) Run(
	ctx context.Context,
	log *logger.Logger,
	taskBus *taskbus.Business,
	eventBus *eventbus.Business,
	noteBus *notebus.Business,
	contextBus *contextbus.Business,
	gapBus *knowledgegapbus.Business,
	args []string,
) error {
	fs := flag.NewFlagSet("gap-backfill", flag.ContinueOnError)
	fs.BoolVar(&cmd.dryRun, "dry-run", false, "analyze gaps but do not create clarification cards")
	fs.StringVar(&cmd.entityType, "entity-type", "", "entity type to backfill: task|note|event|context (empty = all)")
	fs.IntVar(&cmd.limit, "limit", 0, "max entities to process (0 = unlimited)")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}

	// Determine which entity types to process
	entityTypes := []string{"task", "event", "note", "context"}
	if cmd.entityType != "" {
		if !isValidEntityType(cmd.entityType) {
			return fmt.Errorf("invalid entity-type: %s", cmd.entityType)
		}
		entityTypes = []string{cmd.entityType}
	}

	opts := knowledgegapbus.DetectOptions{DryRun: cmd.dryRun}

	log.Info(ctx, "gap-backfill", "status", "starting gap detection backfill",
		"dry_run", cmd.dryRun, "entity_types", strings.Join(entityTypes, ","), "limit", cmd.limit)

	totalProcessed := 0
	totalCardsCreated := 0
	totalSkipped := 0

	// Process each entity type
	for _, entityType := range entityTypes {
		log.Info(ctx, "gap-backfill", "status", fmt.Sprintf("processing %s entities", entityType))

		var (
			cardsCreated int
			skipped      int
			processed    int
			err          error
		)

		switch entityType {
		case "task":
			cardsCreated, skipped, processed, err = cmd.backfillTasks(ctx, log, taskBus, gapBus, opts)
		case "event":
			cardsCreated, skipped, processed, err = cmd.backfillEvents(ctx, log, eventBus, gapBus, opts)
		case "note":
			cardsCreated, skipped, processed, err = cmd.backfillNotes(ctx, log, noteBus, gapBus, opts)
		case "context":
			cardsCreated, skipped, processed, err = cmd.backfillContexts(ctx, log, contextBus, gapBus, opts)
		}

		if err != nil {
			return fmt.Errorf("backfill %s: %w", entityType, err)
		}

		totalProcessed += processed
		totalCardsCreated += cardsCreated
		totalSkipped += skipped

		log.Info(ctx, "gap-backfill", "entity_type", entityType, "processed", processed, "cards_created", cardsCreated, "skipped", skipped)
	}

	log.Info(ctx, "gap-backfill", "status", "complete",
		"total_processed", totalProcessed, "total_cards_created", totalCardsCreated, "total_skipped", totalSkipped)

	return nil
}

func (cmd *GapBackfillCmd) backfillTasks(
	ctx context.Context,
	log *logger.Logger,
	bus *taskbus.Business,
	gapBus *knowledgegapbus.Business,
	opts knowledgegapbus.DetectOptions,
) (int, int, int, error) {
	var cardsCreated, skipped, processed int
	pageNum := 1

	for {
		if cmd.limit > 0 && processed >= cmd.limit {
			break
		}

		pg := page.New(pageNum, 100)
		tasks, err := bus.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pg)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("query tasks: %w", err)
		}

		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			if cmd.limit > 0 && processed >= cmd.limit {
				break
			}

			content := task.Title
			if task.Description != "" {
				content = content + " " + task.Description
			}

			result, err := gapBus.DetectWithOptions(ctx, "task", task.ID, content, opts)
			if err != nil {
				log.Error(ctx, "detect gaps", "entity_type", "task", "entity_id", task.ID, "error", err)
				skipped++
				continue
			}

			cardsCreated += result.CardsCreated
			skipped += result.Skipped
			processed++
		}

		if len(tasks) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	return cardsCreated, skipped, processed, nil
}

func (cmd *GapBackfillCmd) backfillEvents(
	ctx context.Context,
	log *logger.Logger,
	bus *eventbus.Business,
	gapBus *knowledgegapbus.Business,
	opts knowledgegapbus.DetectOptions,
) (int, int, int, error) {
	var cardsCreated, skipped, processed int
	pageNum := 1

	for {
		if cmd.limit > 0 && processed >= cmd.limit {
			break
		}

		pg := page.New(pageNum, 100)
		events, err := bus.Query(ctx, eventbus.QueryFilter{}, eventbus.DefaultOrderBy, pg)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("query events: %w", err)
		}

		if len(events) == 0 {
			break
		}

		for _, event := range events {
			if cmd.limit > 0 && processed >= cmd.limit {
				break
			}

			content := event.Title
			if event.Description != "" {
				content = content + " " + event.Description
			}

			result, err := gapBus.DetectWithOptions(ctx, "event", event.ID, content, opts)
			if err != nil {
				log.Error(ctx, "detect gaps", "entity_type", "event", "entity_id", event.ID, "error", err)
				skipped++
				continue
			}

			cardsCreated += result.CardsCreated
			skipped += result.Skipped
			processed++
		}

		if len(events) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	return cardsCreated, skipped, processed, nil
}

func (cmd *GapBackfillCmd) backfillNotes(
	ctx context.Context,
	log *logger.Logger,
	bus *notebus.Business,
	gapBus *knowledgegapbus.Business,
	opts knowledgegapbus.DetectOptions,
) (int, int, int, error) {
	var cardsCreated, skipped, processed int
	pageNum := 1

	for {
		if cmd.limit > 0 && processed >= cmd.limit {
			break
		}

		pg := page.New(pageNum, 100)
		notes, err := bus.Query(ctx, notebus.QueryFilter{}, notebus.DefaultOrderBy, pg)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("query notes: %w", err)
		}

		if len(notes) == 0 {
			break
		}

		for _, note := range notes {
			if cmd.limit > 0 && processed >= cmd.limit {
				break
			}

			result, err := gapBus.DetectWithOptions(ctx, "note", note.ID, note.Content, opts)
			if err != nil {
				log.Error(ctx, "detect gaps", "entity_type", "note", "entity_id", note.ID, "error", err)
				skipped++
				continue
			}

			cardsCreated += result.CardsCreated
			skipped += result.Skipped
			processed++
		}

		if len(notes) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	return cardsCreated, skipped, processed, nil
}

func (cmd *GapBackfillCmd) backfillContexts(
	ctx context.Context,
	log *logger.Logger,
	bus *contextbus.Business,
	gapBus *knowledgegapbus.Business,
	opts knowledgegapbus.DetectOptions,
) (int, int, int, error) {
	var cardsCreated, skipped, processed int
	pageNum := 1

	for {
		if cmd.limit > 0 && processed >= cmd.limit {
			break
		}

		pg := page.New(pageNum, 100)
		contexts, err := bus.Query(ctx, contextbus.QueryFilter{}, contextbus.DefaultOrderBy, pg)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("query contexts: %w", err)
		}

		if len(contexts) == 0 {
			break
		}

		for _, ctx_item := range contexts {
			if cmd.limit > 0 && processed >= cmd.limit {
				break
			}

			content := ctx_item.Title
			if ctx_item.Description != "" {
				content = content + " " + ctx_item.Description
			}

			result, err := gapBus.DetectWithOptions(ctx, "context", ctx_item.ID, content, opts)
			if err != nil {
				log.Error(ctx, "detect gaps", "entity_type", "context", "entity_id", ctx_item.ID, "error", err)
				skipped++
				continue
			}

			cardsCreated += result.CardsCreated
			skipped += result.Skipped
			processed++
		}

		if len(contexts) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	return cardsCreated, skipped, processed, nil
}

func isValidEntityType(entityType string) bool {
	switch entityType {
	case "task", "event", "note", "context":
		return true
	default:
		return false
	}
}
