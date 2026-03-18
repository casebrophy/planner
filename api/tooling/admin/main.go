package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ardanlabs/conf"

	"github.com/casebrophy/planner/business/sdk/migrate"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
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
		fmt.Println("Commands: migrate, seed")
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

	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}

	return nil
}
