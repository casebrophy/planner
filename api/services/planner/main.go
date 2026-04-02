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

	"github.com/casebrophy/planner/app/domain/checkapp"
	"github.com/casebrophy/planner/app/domain/clarificationapp"
	"github.com/casebrophy/planner/app/domain/contextapp"
	"github.com/casebrophy/planner/app/domain/dailyplanapp"
	"github.com/casebrophy/planner/app/domain/emailapp"
	"github.com/casebrophy/planner/app/domain/eventapp"
	"github.com/casebrophy/planner/app/domain/scheduleapp"
	"github.com/casebrophy/planner/app/domain/mcpapp"
	"github.com/casebrophy/planner/app/domain/observationapp"
	"github.com/casebrophy/planner/app/domain/rawinputapp"
	"github.com/casebrophy/planner/app/domain/tagapp"
	"github.com/casebrophy/planner/app/domain/taskapp"
	"github.com/casebrophy/planner/app/domain/threadapp"
	"github.com/casebrophy/planner/app/domain/timeblockapp"
	"github.com/casebrophy/planner/app/domain/transactionapp"
	"github.com/casebrophy/planner/app/domain/voiceingestapp"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/domain/emailbus/stores/emaildb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/google/uuid"
	"github.com/casebrophy/planner/business/domain/dailyplanbus"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/generator"
	"github.com/casebrophy/planner/business/domain/dailyplanbus/stores/dailyplandb"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/domain/inactivitybus/stores/inactivitydb"
	"github.com/casebrophy/planner/business/domain/ingestbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/foundation/claudecli"
	"github.com/casebrophy/planner/business/domain/rawinputbus/stores/rawinputdb"
	"github.com/casebrophy/planner/business/domain/smtpbus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
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
			WriteTimeout    time.Duration `conf:"default:10s"`
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
			CLIPath string `conf:"default:claude"`
			Models  string `conf:"default:haiku,sonnet,opus"`
		}
		DailyPlan struct {
			Time    string `conf:"default:07:00"`
			Enabled bool   `conf:"default:true"`
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

	inactStore := inactivitydb.NewStore(log, db)
	inactBus := inactivitybus.NewBusiness(log, inactStore, clarBus)

	// Daily plan generation dependencies
	dpStore := dailyplandb.NewStore(log, db)
	dpBus := dailyplanbus.NewBusiness(log, dpStore)

	taskStore := taskdb.NewStore(log, db)
	taskBus := taskbus.NewBusiness(log, taskStore)

	ctxStore := contextdb.NewStore(log, db)
	ctxBus := contextbus.NewBusiness(log, ctxStore)

	evtStore := eventdb.NewStore(log, db)
	evtBus := eventbus.NewBusiness(log, evtStore)

	// -------------------------------------------------------------------------
	// Build Handler

	log.Info(ctx, "startup", "status", "initializing api")

	cli := claudecli.NewClient(log, cfg.Claude.CLIPath, strings.Split(cfg.Claude.Models, ","))

	muxCfg := mux.Config{
		Log:         log,
		DB:          db,
		APIKey:      cfg.Auth.APIKey,
		ClaudeCLI:   cli,
		CORSOrigins: strings.Split(cfg.Web.CORSOrigins, ","),
	}

	handler := mux.WebAPI(muxCfg,
		checkapp.Routes{},
		taskapp.Routes{},
		contextapp.Routes{},
		tagapp.Routes{},
		rawinputapp.Routes{},
		emailapp.Routes{},
		transactionapp.Routes{},
		clarificationapp.Routes{},
		threadapp.Routes{},
		observationapp.Routes{},
		voiceingestapp.Routes{},
		eventapp.Routes{},
		dailyplanapp.Routes{},
		timeblockapp.Routes{},
		scheduleapp.Routes{},
		mcpapp.Routes{},
	)

	// -------------------------------------------------------------------------
	// SMTP Server (Email Ingestion)

	var smtpSrv *smtpbus.Server
	if cfg.SMTP.Enabled {
		log.Info(ctx, "startup", "status", "initializing smtp server")

		riStore := rawinputdb.NewStore(log, db)
		riBus := rawinputbus.NewBusiness(log, riStore)

		emStore := emaildb.NewStore(log, db)
		emBus := emailbus.NewBusiness(log, emStore)

		tStore := taskdb.NewStore(log, db)
		tBus := taskbus.NewBusiness(log, tStore)

		cStore := contextdb.NewStore(log, db)
		cBus := contextbus.NewBusiness(log, cStore)

		eStore := eventdb.NewStore(log, db)
		eBus := eventbus.NewBusiness(log, eStore)

		ext := extractor.NewClaudeCodeExtractor(cli)
		igBus := ingestbus.NewBusiness(log, riBus, emBus, tBus, cBus, clarBus, eBus, ext)

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

	// Inactivity detection: runs every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
				if err := inactBus.CheckAll(jobCtx); err != nil {
					log.Error(jobCtx, "inactivity", "msg", "inactivity check failed", "error", err)
				}
			}
		}
	}()

	// Unsnooze expired clarifications: runs every 5 minutes
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-jobCtx.Done():
				return
			case <-ticker.C:
				n, err := clarBus.UnsnoozeExpired(jobCtx)
				if err != nil {
					log.Error(jobCtx, "unsnooze", "msg", "unsnooze expired failed", "error", err)
				} else if n > 0 {
					log.Info(jobCtx, "unsnooze", "msg", "unsnoozed expired items", "count", n)
				}
			}
		}
	}()

	// Daily plan generation: runs once per day at configured time
	if cfg.DailyPlan.Enabled {
		gen := generator.NewGenerator(cli)
		go func() {
			ticker := time.NewTicker(1 * time.Minute)
			defer ticker.Stop()

			var lastGenDate string

			for {
				select {
				case <-jobCtx.Done():
					return
				case <-ticker.C:
					now := time.Now()
					todayStr := now.Format("2006-01-02")
					timeStr := now.Format("15:04")

					if timeStr != cfg.DailyPlan.Time || lastGenDate == todayStr {
						continue
					}
					lastGenDate = todayStr
					log.Info(jobCtx, "daily-plan", "msg", "generating morning plan", "date", todayStr)

					// Fetch open tasks
					allTasks, err := taskBus.Query(jobCtx, taskbus.QueryFilter{}, taskbus.DefaultOrderBy, page.MustParse("1", "200"))
					if err != nil {
						log.Error(jobCtx, "daily-plan", "msg", "failed to fetch tasks", "error", err)
						continue
					}

					// Build context title lookup
					activeStatus := contextbus.Active
					contexts, _ := ctxBus.Query(jobCtx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.MustParse("1", "100"))
					ctxNames := make(map[uuid.UUID]string, len(contexts))
					for _, c := range contexts {
						ctxNames[c.ID] = c.Title
					}

					var taskRefs []generator.TaskRef
					for _, t := range allTasks {
						if t.Status.String() != "todo" && t.Status.String() != "in_progress" {
							continue
						}
						ref := generator.TaskRef{
							ID:       t.ID.String(),
							Title:    t.Title,
							Priority: t.Priority.String(),
							Energy:   t.Energy.String(),
							Status:   t.Status.String(),
						}
						if t.DurationMin != nil {
							ref.DurationMin = t.DurationMin
						}
						if t.DueDate != nil {
							s := t.DueDate.Format(time.RFC3339)
							ref.DueDate = &s
						}
						if t.ContextID != nil {
							if name, ok := ctxNames[*t.ContextID]; ok {
								ref.Context = &name
							}
						}
						taskRefs = append(taskRefs, ref)
					}

					// Fetch today's events
					today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
					tomorrow := today.Add(24 * time.Hour)
					events, err := evtBus.Query(jobCtx, eventbus.QueryFilter{DateFrom: &today, DateTo: &tomorrow}, eventbus.DefaultOrderBy, page.MustParse("1", "50"))
					if err != nil {
						log.Error(jobCtx, "daily-plan", "msg", "failed to fetch events", "error", err)
						continue
					}

					var eventRefs []generator.EventRef
					for _, e := range events {
						ref := generator.EventRef{
							ID:       e.ID.String(),
							Title:    e.Title,
							StartsAt: e.StartsAt.Format(time.RFC3339),
							EndsAt:   e.EndsAt.Format(time.RFC3339),
							AllDay:   e.AllDay,
						}
						if e.Location != nil {
							ref.Location = e.Location
						}
						eventRefs = append(eventRefs, ref)
					}

					// Generate plan
					output, err := gen.Generate(jobCtx, taskRefs, eventRefs, nil)
					if err != nil {
						log.Error(jobCtx, "daily-plan", "msg", "plan generation failed", "error", err)
						continue
					}

					// Store plan
					plan, err := dpBus.Create(jobCtx, dailyplanbus.NewDailyPlan{
						PlanDate:   today,
						Generation: 1,
						ModelUsed:  "haiku",
					})
					if err != nil {
						log.Error(jobCtx, "daily-plan", "msg", "failed to create plan", "error", err)
						continue
					}

					itemCount := 0
					for gi, group := range output.Groups {
						for ii, item := range group.Items {
							taskID, err := uuid.Parse(item.TaskID)
							if err != nil {
								continue
							}
							reason := item.PriorityReason
							dur := item.AIDurationMin
							if _, err := dpBus.AddItem(jobCtx, dailyplanbus.NewDailyPlanItem{
								PlanID:           plan.ID,
								TaskID:           taskID,
								Position:         ii,
								GroupName:        group.Name,
								GroupPosition:    gi,
								AIDurationMin:    &dur,
								AIPriorityReason: &reason,
							}); err != nil {
								log.Error(jobCtx, "daily-plan", "msg", "failed to add item", "error", err)
								continue
							}
							itemCount++
						}
					}

					log.Info(jobCtx, "daily-plan", "msg", "morning plan generated", "items", itemCount, "groups", len(output.Groups))
				}
			}
		}()
	}

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
