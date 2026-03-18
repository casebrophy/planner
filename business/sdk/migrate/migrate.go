package migrate

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/ardanlabs/darwin/v3"
	"github.com/ardanlabs/darwin/v3/dialects/postgres"
	"github.com/ardanlabs/darwin/v3/drivers/generic"
	"github.com/jmoiron/sqlx"
)

//go:embed sql/migrate.sql
var migrateSQL string

//go:embed sql/seed.sql
var seedSQL string

func Migrate(ctx context.Context, db *sqlx.DB) error {
	driver, err := generic.New(db.DB, postgres.Dialect{})
	if err != nil {
		return fmt.Errorf("create darwin driver: %w", err)
	}

	d := darwin.New(driver, darwin.ParseMigrations(migrateSQL))
	if err := d.Migrate(); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	return nil
}

func Seed(ctx context.Context, db *sqlx.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("seed begin: %w", err)
	}

	if _, err := tx.Exec(seedSQL); err != nil {
		tx.Rollback()
		return fmt.Errorf("seed exec: %w", err)
	}

	return tx.Commit()
}
