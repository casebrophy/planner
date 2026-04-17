package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ardanlabs/conf"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/api/tooling/admin/commands"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus/stores/embeddingdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/sdk/migrate"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/embed"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/ollamaclient"
)

func main() {
	log := logger.New(os.Stdout, logger.LevelInfo, "admin")

	if err := run(log); err != nil {
		log.Error(context.Background(), "error", "msg", err)
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	cfg := struct {
		DB sqldb.Config
	}{}

	const prefix = "PLANNER"
	if err := conf.Parse(os.Args[1:], prefix, &cfg); err != nil {
		if err == conf.ErrHelpWanted {
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	db, err := sqldb.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()

	if len(os.Args) < 2 {
		fmt.Println("Usage: admin <command>")
		fmt.Println("Commands: migrate, seed, backfill-embeddings, gap-backfill")
		return nil
	}

	switch os.Args[1] {
	case "migrate":
		log.Info(ctx, "admin", "status", "running migrations")
		if err := migrate.Migrate(ctx, db); err != nil {
			return fmt.Errorf("migrate: %w", err)
		}
		log.Info(ctx, "admin", "status", "migrations complete")

	case "seed":
		log.Info(ctx, "admin", "status", "seeding database")
		if err := migrate.Seed(ctx, db); err != nil {
			return fmt.Errorf("seed: %w", err)
		}
		log.Info(ctx, "admin", "status", "seeding complete")

	case "backfill-embeddings":
		log.Info(ctx, "admin", "status", "backfilling embeddings")
		if err := backfillEmbeddings(ctx, log, db); err != nil {
			return fmt.Errorf("backfill: %w", err)
		}
		log.Info(ctx, "admin", "status", "backfill complete")

	case "gap-backfill":
		log.Info(ctx, "admin", "status", "backfilling knowledge gaps")
		if err := gapBackfill(ctx, log, db); err != nil {
			return fmt.Errorf("gap-backfill: %w", err)
		}
		log.Info(ctx, "admin", "status", "gap-backfill complete")

	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}

	return nil
}

func backfillEmbeddings(ctx context.Context, log *logger.Logger, db *sqlx.DB) error {
	// Wire up the buses
	taskStore := taskdb.NewStore(log, db)
	depStore := taskdb.NewDependencyStore(log, db)
	taskBus := taskbus.NewBusiness(log, taskStore, depStore)

	evtStore := eventdb.NewStore(log, db)
	evtBus := eventbus.NewBusiness(log, evtStore)

	noteStore := notedb.NewStore(log, db)
	noteBus := notebus.NewBusiness(log, noteStore)

	embStore := embeddingdb.NewStore(log, db)
	// For backfill, embedder must be configured (not nil)
	// Using Ollama embedder for local-only embedding
	ollamaURL := os.Getenv("PLANNER_OLLAMA_URL")
	if ollamaURL == "" {
		ollamaURL = "http://localhost:11434"
	}
	embedModel := os.Getenv("PLANNER_OLLAMA_EMBED_MODEL")
	if embedModel == "" {
		embedModel = "qwen3-embedding:0.6b"
	}
	client := ollamaclient.New(ollamaclient.Config{BaseURL: ollamaURL})
	defer client.Close()
	embedder := embed.NewOllamaEmbedder(client, embedModel, 1024)
	embBus := embeddingbus.NewBusiness(log, embStore, embedder)

	// Track metrics
	var totalEntities, embeddedCount, skippedCount, errorCount int

	// Backfill tasks
	log.Info(ctx, "backfill", "status", "processing tasks")
	pageNum := 1
	for {
		pg := page.New(pageNum, 100)
		tasks, err := taskBus.Query(ctx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, pg)
		if err != nil {
			return fmt.Errorf("query tasks: %w", err)
		}
		if len(tasks) == 0 {
			break
		}

		for _, task := range tasks {
			totalEntities++
			exists, err := embStore.ExistsBySource(ctx, "task", task.ID)
			if err != nil {
				log.Error(ctx, "check embedding", "source_type", "task", "source_id", task.ID, "error", err)
				errorCount++
				continue
			}

			if exists {
				skippedCount++
				continue
			}

			// Embed task (title + description)
			content := task.Title
			if task.Description != "" {
				content = content + " " + task.Description
			}

			if err := embBus.EmbedAndStore(ctx, "task", task.ID, content); err != nil {
				log.Error(ctx, "embed task", "task_id", task.ID, "error", err)
				errorCount++
				continue
			}

			embeddedCount++
			if embeddedCount%10 == 0 {
				time.Sleep(100 * time.Millisecond) // Brief pause between batches
				log.Info(ctx, "backfill", "embedded", embeddedCount)
			}
		}

		if len(tasks) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	// Backfill events
	log.Info(ctx, "backfill", "status", "processing events")
	pageNum = 1
	for {
		pg := page.New(pageNum, 100)
		events, err := evtBus.Query(ctx, eventbus.QueryFilter{}, eventbus.DefaultOrderBy, pg)
		if err != nil {
			return fmt.Errorf("query events: %w", err)
		}
		if len(events) == 0 {
			break
		}

		for _, evt := range events {
			totalEntities++
			exists, err := embStore.ExistsBySource(ctx, "event", evt.ID)
			if err != nil {
				log.Error(ctx, "check embedding", "source_type", "event", "source_id", evt.ID, "error", err)
				errorCount++
				continue
			}

			if exists {
				skippedCount++
				continue
			}

			content := evt.Title
			if evt.Description != "" {
				content = content + " " + evt.Description
			}

			if err := embBus.EmbedAndStore(ctx, "event", evt.ID, content); err != nil {
				log.Error(ctx, "embed event", "event_id", evt.ID, "error", err)
				errorCount++
				continue
			}

			embeddedCount++
			if embeddedCount%10 == 0 {
				time.Sleep(100 * time.Millisecond)
				log.Info(ctx, "backfill", "embedded", embeddedCount)
			}
		}

		if len(events) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	// Backfill notes
	log.Info(ctx, "backfill", "status", "processing notes")
	pageNum = 1
	for {
		pg := page.New(pageNum, 100)
		notes, err := noteBus.Query(ctx, notebus.QueryFilter{}, notebus.DefaultOrderBy, pg)
		if err != nil {
			return fmt.Errorf("query notes: %w", err)
		}
		if len(notes) == 0 {
			break
		}

		for _, note := range notes {
			totalEntities++
			exists, err := embStore.ExistsBySource(ctx, "note", note.ID)
			if err != nil {
				log.Error(ctx, "check embedding", "source_type", "note", "source_id", note.ID, "error", err)
				errorCount++
				continue
			}

			if exists {
				skippedCount++
				continue
			}

			if err := embBus.EmbedAndStore(ctx, "note", note.ID, note.Content); err != nil {
				log.Error(ctx, "embed note", "note_id", note.ID, "error", err)
				errorCount++
				continue
			}

			embeddedCount++
			if embeddedCount%10 == 0 {
				time.Sleep(100 * time.Millisecond)
				log.Info(ctx, "backfill", "embedded", embeddedCount)
			}
		}

		if len(notes) < pg.RowsPerPage() {
			break
		}
		pageNum++
	}

	log.Info(ctx, "backfill", "complete", "total", totalEntities, "embedded", embeddedCount, "skipped", skippedCount, "errors", errorCount)
	return nil
}

func gapBackfill(ctx context.Context, log *logger.Logger, db *sqlx.DB) error {
	// Wire up the buses
	taskStore := taskdb.NewStore(log, db)
	depStore := taskdb.NewDependencyStore(log, db)
	taskBus := taskbus.NewBusiness(log, taskStore, depStore)

	evtStore := eventdb.NewStore(log, db)
	evtBus := eventbus.NewBusiness(log, evtStore)

	noteStore := notedb.NewStore(log, db)
	noteBus := notebus.NewBusiness(log, noteStore)

	ctxStore := contextdb.NewStore(log, db)
	ctxBus := contextbus.NewBusiness(log, ctxStore)

	clarificationStore := clarificationdb.NewStore(log, db)
	clarificationBus := clarificationbus.NewBusiness(log, clarificationStore)

	embStore := embeddingdb.NewStore(log, db)
	embBus := embeddingbus.NewBusiness(log, embStore, nil)

	// Create a gap analyzer (stub for admin tool - errors don't block backfill)
	// In a real deployment, this would be the full extractor adapter
	// For now, we just need something that satisfies the interface
	gapBus := knowledgegapbus.New(log, clarificationBus, embBus, &stubGapAnalyzer{}, knowledgegapbus.Config{})

	cmd := &commands.GapBackfillCmd{}
	return cmd.Run(ctx, log, taskBus, evtBus, noteBus, ctxBus, gapBus, os.Args[2:])
}

type stubGapAnalyzer struct{}

func (s *stubGapAnalyzer) AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []knowledgegapbus.RelatedEntitySummary) (knowledgegapbus.GapAnalysis, error) {
	return knowledgegapbus.GapAnalysis{Gaps: []knowledgegapbus.GapCandidate{}}, nil
}
