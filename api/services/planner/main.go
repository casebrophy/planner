package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ardanlabs/conf"

	"github.com/casebrophy/planner/api/services/planner/jobs"
	"github.com/casebrophy/planner/app/domain/activitylogapp"
	"github.com/casebrophy/planner/app/domain/checkapp"
	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/domain/classifyapp"
	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/domain/correctionapp"
	"github.com/casebrophy/planner/app/domain/dailyplanapp"
	"github.com/casebrophy/planner/app/domain/emailapp"
	"github.com/casebrophy/planner/app/domain/entitylinkapp"
	"github.com/casebrophy/planner/app/domain/eventapp"
	"github.com/casebrophy/planner/app/domain/mcpapp"
	"github.com/casebrophy/planner/app/domain/noteapp"
	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/domain/ollamaapp"
	"github.com/casebrophy/planner/app/domain/rawinputapp"
	"github.com/casebrophy/planner/app/domain/reingestapp"
	"github.com/casebrophy/planner/app/domain/scheduleapp"
	"github.com/casebrophy/planner/app/domain/serverapp"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/domain/timeblockapp"
	"github.com/casebrophy/planner/app/domain/splitapp"
	"github.com/casebrophy/planner/app/domain/transactionapp"
	"github.com/casebrophy/planner/app/domain/voiceingestapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/stores/dailyplandb"
	"github.com/casebrophy/planner/business/domain/debriefbus"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/embeddingbus"
	"github.com/casebrophy/planner/business/domain/embeddingbus/stores/embeddingdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/inactivitybus/stores/inactivitydb"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/knowledgegapbus"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/smtpbus"
	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/domain/tagbus/stores/tagdb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/domain/threadbus"
	"github.com/casebrophy/planner/business/domain/threadbus/stores/threaddb"
	"github.com/casebrophy/planner/business/sdk/sanitize"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/sdk/worker"
	"github.com/casebrophy/planner/business/types/gapcategory"
	"github.com/casebrophy/planner/foundation/claudecli"
	"github.com/casebrophy/planner/foundation/embed"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/ollamaclient"
)

var build = "develop"

func main() {
	log := logger.New(os.Stdout, logger.LevelInfo, "planner")

	if err := run(log); err != nil {
		log.Error(context.Background(), "startup", "error", err)
		os.Exit(1)
	}
}

func run(log *logger.Logger) error {
	// -------------------------------------------------------------------------
	// Configuration

	cfg := struct {
		Web struct {
			APIHost         string        `conf:"default:0.0.0.0:8080"`
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:180s"`
			IdleTimeout     time.Duration `conf:"default:120s"`
			ShutdownTimeout time.Duration `conf:"default:20s"`
			CORSOrigins     string        `conf:"default:*"`
		}
		DB   sqldb.Config
		Auth struct {
			APIKey string `conf:"mask"`
		}
		SMTP struct {
			Addr    string `conf:"default::2525"`
			Domain  string `conf:"default:localhost"`
			Enabled bool   `conf:"default:false"`
		}
		Claude struct {
			Models string `conf:"default:haiku,sonnet,opus"`
		}
		DailyPlan struct {
			Time     string `conf:"default:07:00"`
			Enabled  bool   `conf:"default:true"`
			Timezone string `conf:"default:America/Denver"`
		}
		Sidecar struct {
			URL string
		}
		Ollama struct {
			URL        string
			Model      string        `conf:"default:qwen3.5:0.8b"`
			EmbedModel string        `conf:"default:qwen3-embedding:0.6b"`
			Enabled    bool          `conf:"default:true"`
			Timeout    time.Duration `conf:"default:180s"`
			QueueSize  int           `conf:"default:64"`
		}
	}{}

	const prefix = "PLANNER"
	err := conf.Parse(os.Args[1:], prefix, &cfg)
	if err != nil {
		if err == conf.ErrHelpWanted {
			return nil
		}
		return fmt.Errorf("parsing config: %w", err)
	}

	userTZ, err := time.LoadLocation(cfg.DailyPlan.Timezone)
	if err != nil {
		return fmt.Errorf("loading timezone %q: %w", cfg.DailyPlan.Timezone, err)
	}

	// -------------------------------------------------------------------------
	// App Starting

	log.Info(context.Background(), "starting service", "version", build)

	// -------------------------------------------------------------------------
	// Database

	log.Info(context.Background(), "startup", "status", "initializing database")

	db, err := sqldb.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}
	defer db.Close()

	ctx := context.Background()
	if err := sqldb.StatusCheck(ctx, db); err != nil {
		return fmt.Errorf("db status check: %w", err)
	}

	// -------------------------------------------------------------------------
	// Business Layer (top-level, shared across consumers)

	clarStore := clarificationdb.NewStore(log, db)
	clarBus := clarificationbus.NewBusiness(log, clarStore)

	threadStore := threaddb.NewStore(log, db)
	threadBus := threadbus.NewBusiness(log, threadStore)

	debriefBus := debriefbus.NewBusiness(log, clarBus, threadBus)

	inactStore := inactivitydb.NewStore(log, db)
	inactBus := inactivitybus.NewBusiness(log, inactStore, clarBus)

	// Daily plan generation dependencies
	dpStore := dailyplandb.NewStore(log, db)
	dpBus := dailyplanbus.NewBusiness(log, dpStore)

	taskStore := taskdb.NewStore(log, db)
	depStore := taskdb.NewDependencyStore(log, db)
	taskBus := taskbus.NewBusiness(log, taskStore, depStore)

	ctxStore := contextdb.NewStore(log, db)
	ctxBus := contextbus.NewBusiness(log, ctxStore)

	evtStore := eventdb.NewStore(log, db)
	evtBus := eventbus.NewBusiness(log, evtStore)

	riStore := rawinputdb.NewStore(log, db)
	riBus := rawinputbus.NewBusiness(log, riStore)

	emStore := emaildb.NewStore(log, db)
	emBus := emailbus.NewBusiness(log, emStore)

	noteStore := notedb.NewStore(log, db)
	noteBus := notebus.NewBusiness(log, noteStore)
	tagStore := tagdb.New(log, db)
	tgBus := tagbus.NewBusiness(log, tagStore)

	// -------------------------------------------------------------------------
	// Build Handler

	log.Info(ctx, "startup", "status", "initializing api")

	cli := claudecli.NewClient(log, strings.Split(cfg.Claude.Models, ","), cfg.Sidecar.URL, cfg.Auth.APIKey)
	log.Info(ctx, "startup", "status", "inference routed via sidecar", "url", cfg.Sidecar.URL)

	claudeExt := extractor.NewClaudeCodeExtractor(cli)

	ollamaEnabled := cfg.Ollama.URL != "" && cfg.Ollama.Enabled
	log.Info(ctx, "startup", "ollama_enabled", ollamaEnabled, "ollama_url", cfg.Ollama.URL, "embed_model", cfg.Ollama.EmbedModel)

	var ollamaClient *ollamaclient.Client
	if ollamaEnabled {
		ollamaClient = ollamaclient.New(ollamaclient.Config{
			BaseURL:   cfg.Ollama.URL,
			Timeout:   cfg.Ollama.Timeout,
			QueueSize: cfg.Ollama.QueueSize,
		})
		defer ollamaClient.Close()
	}

	var ext extractor.Extractor
	if ollamaEnabled {
		ollamaExt := extractor.NewOllamaExtractor(ollamaClient, cfg.Ollama.Model)
		failover := extractor.NewFailoverExtractor(log, claudeExt, ollamaExt)
		ext = extractor.NewTieredRouter(log, failover, ollamaExt)
	} else {
		ext = claudeExt
	}
	igBus := ingestbus.NewBusiness(log, riBus, emBus, taskBus, ctxBus, clarBus, evtBus, ext, noteBus, tgBus)

	embStore := embeddingdb.NewStore(log, db)
	var embedder embed.Embedder
	if ollamaEnabled {
		embedder = embed.NewOllamaEmbedder(ollamaClient, cfg.Ollama.EmbedModel, 1024)
	}
	embBus := embeddingbus.NewBusiness(log, embStore, embedder)

	clarStoreGap := clarificationdb.NewStore(log, db)
	clarBusGap := clarificationbus.NewBusiness(log, clarStoreGap)
	gapBus := knowledgegapbus.New(log, clarBusGap, embBus, &extractorGapAdapter{ext: ext}, knowledgegapbus.Config{})
	igBus.WithEmbedder(embBus)
	igBus.WithGapDetector(gapBus)

	muxCfg := mux.Config{
		Log:              log,
		DB:               db,
		APIKey:           cfg.Auth.APIKey,
		ClaudeCLI:        cli,
		CORSOrigins:      strings.Split(cfg.Web.CORSOrigins, ","),
		SidecarURL:       cfg.Sidecar.URL,
		OllamaURL:        cfg.Ollama.URL,
		OllamaModel:      cfg.Ollama.Model,
		OllamaEmbedModel: cfg.Ollama.EmbedModel,
		OllamaEnabled:    ollamaEnabled,
		OllamaClient:     ollamaClient,
		Extractor:        ext,
		EmbeddingBus:     embBus,
		KnowledgeGapBus:  gapBus,
		UserTimezone:     userTZ,
	}

	handler := mux.WebAPI(muxCfg,
		checkapp.Routes{},
		taskapp.Routes{},
		contextapp.Routes{},
		tagapp.Routes{},
		noteapp.Routes{},
		rawinputapp.Routes{},
		emailapp.Routes{},
		transactionapp.Routes{},
		splitapp.Routes{},
		clarificationapp.Routes{},
		threadapp.Routes{},
		observationapp.Routes{},
		voiceingestapp.Routes{},
		eventapp.Routes{},
		dailyplanapp.Routes{},
		timeblockapp.Routes{},
		scheduleapp.Routes{},
		mcpapp.Routes{},
		serverapp.Routes{},
		activitylogapp.Routes{},
		classifyapp.Routes{},
		entitylinkapp.Routes{},
		correctionapp.Routes{},
		ollamaapp.Routes{},
		reingestapp.Routes{},
	)

	// -------------------------------------------------------------------------
	// SMTP Server (Email Ingestion)

	var smtpSrv *smtpbus.Server
	if cfg.SMTP.Enabled {
		log.Info(ctx, "startup", "status", "initializing smtp server")
		smtpSrv = smtpbus.NewServer(log, igBus, smtpbus.Config{
			Addr:   cfg.SMTP.Addr,
			Domain: cfg.SMTP.Domain,
		})
	}

	// -------------------------------------------------------------------------
	// Start Server

	api := http.Server{
		Addr:         cfg.Web.APIHost,
		Handler:      handler,
		ReadTimeout:  cfg.Web.ReadTimeout,
		WriteTimeout: cfg.Web.WriteTimeout,
		IdleTimeout:  cfg.Web.IdleTimeout,
		ErrorLog:     logger.NewStdLogger(log, logger.LevelError),
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info(ctx, "startup", "status", "api router started", "host", api.Addr)
		serverErrors <- api.ListenAndServe()
	}()

	if smtpSrv != nil {
		go func() {
			if err := smtpSrv.ListenAndServe(); err != nil {
				log.Error(ctx, "smtp", "msg", "smtp server error", "error", err)
			}
		}()
	}

	// -------------------------------------------------------------------------
	// Background Jobs

	jobCtx, jobCancel := context.WithCancel(ctx)
	defer jobCancel()

	bgJobs := []jobs.Job{
		jobs.InactivityJob{Log: log, Checker: inactBus},
		jobs.UnsnoozeJob{Log: log, Bus: clarBus},
		jobs.RawInputRecoveryJob{Log: log, Bus: riBus},
		jobs.WeeklyReviewJob{Log: log, TaskBus: taskBus, DebriefBus: debriefBus},
		worker.NewIngestWorker(log, riBus, igBus),
	}
	if cfg.DailyPlan.Enabled {
		gen := generator.NewGenerator(cli)
		bgJobs = append(bgJobs, jobs.DailyPlanJob{
			Log:       log,
			Generator: gen,
			TaskBus:   taskBus,
			CtxBus:    ctxBus,
			EvtBus:    evtBus,
			DpBus:     dpBus,
			UserTZ:    userTZ,
			FireAt:    cfg.DailyPlan.Time,
		})
	}

	go jobs.RunAll(jobCtx, log, bgJobs...)

	// -------------------------------------------------------------------------
	// Shutdown

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)

	case sig := <-shutdown:
		log.Info(ctx, "shutdown", "status", "shutdown started", "signal", sig)
		defer log.Info(ctx, "shutdown", "status", "shutdown complete", "signal", sig)

		// Stop background jobs
		jobCancel()

		ctx, cancel := context.WithTimeout(ctx, cfg.Web.ShutdownTimeout)
		defer cancel()

		if smtpSrv != nil {
			if err := smtpSrv.Close(); err != nil {
				log.Error(ctx, "shutdown", "msg", "smtp shutdown error", "error", err)
			}
		}

		if err := api.Shutdown(ctx); err != nil {
			api.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// extractorGapAdapter adapts extractor.Extractor to knowledgegapbus.GapAnalyzer.
type extractorGapAdapter struct {
	ext extractor.Extractor
}

func (a *extractorGapAdapter) AnalyzeGaps(ctx context.Context, entityContent string, relatedSummaries []knowledgegapbus.RelatedEntitySummary) (knowledgegapbus.GapAnalysis, error) {
	// Sanitize entityContent before sending to Claude (belt-and-suspenders for incidental PII)
	entityContent = sanitize.Sanitize(entityContent).Text

	relatedEntities := make([]extractor.RelatedEntity, len(relatedSummaries))
	for i, s := range relatedSummaries {
		relatedEntities[i] = extractor.RelatedEntity{
			ID:         s.SourceID,
			SourceType: s.SourceType,
			Content:    s.Content,
		}
	}
	result, err := a.ext.AnalyzeGaps(ctx, "", entityContent, relatedEntities)
	if err != nil {
		return knowledgegapbus.GapAnalysis{}, err
	}
	var gaps []knowledgegapbus.GapCandidate
	for _, g := range result.Gaps {
		cat, err := gapcategory.Parse(g.Category)
		if err != nil {
			continue
		}
		gaps = append(gaps, knowledgegapbus.GapCandidate{
			Category:   cat,
			Question:   g.Question,
			Reasoning:  g.Reasoning,
			Confidence: g.Confidence,
			RelatedIDs: g.RelatedIDs,
			Options:    g.Options,
		})
	}
	return knowledgegapbus.GapAnalysis{Gaps: gaps}, nil
}
