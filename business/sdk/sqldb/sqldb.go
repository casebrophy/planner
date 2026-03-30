package sqldb

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/foundation/logger"
)

var (
	ErrDBNotFound        = sql.ErrNoRows
	ErrDBDuplicatedEntry = errors.New("duplicated entry")
)

type Config struct {
	Host       string `conf:"default:localhost"`
	Port       int    `conf:"default:5432"`
	User       string `conf:"default:planner"`
	Password   string `conf:"default:planner,mask"`
	Name       string `conf:"default:planner"`
	DisableTLS bool   `conf:"default:true"`
}

func Open(cfg Config) (*sqlx.DB, error) {
	sslMode := "require"
	if cfg.DisableTLS {
		sslMode = "disable"
	}

	q := make(url.Values)
	q.Set("sslmode", sslMode)
	q.Set("timezone", "utc")

	u := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(cfg.User, cfg.Password),
		Host:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Path:     cfg.Name,
		RawQuery: q.Encode(),
	}

	db, err := sqlx.Open("pgx", u.String())
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)

	return db, nil
}

func StatusCheck(ctx context.Context, db *sqlx.DB) error {
	return db.PingContext(ctx)
}

func NamedExecContext(ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any) error {
	q, args, err := sqlx.Named(query, data)
	if err != nil {
		return fmt.Errorf("named exec: %w", err)
	}

	q = sqlx.Rebind(sqlx.DOLLAR, q)

	if _, err := db.ExecContext(ctx, q, args...); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrDBDuplicatedEntry
		}
		return fmt.Errorf("exec: %w", err)
	}

	return nil
}

// NamedQueryStruct executes a named query and scans the result into a struct.
// Returns ErrDBNotFound if no rows are found.
func NamedQueryStruct[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any, dest *T) error {
	q, args, err := sqlx.Named(query, data)
	if err != nil {
		return fmt.Errorf("named query struct: %w", err)
	}

	q = sqlx.Rebind(sqlx.DOLLAR, q)

	row := db.QueryRowxContext(ctx, q, args...)
	if err := row.StructScan(dest); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrDBNotFound
		}
		return fmt.Errorf("query struct: %w", err)
	}

	return nil
}

// NamedQuerySlice executes a named query and scans all results into a slice.
func NamedQuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string, data any) ([]T, error) {
	q, args, err := sqlx.Named(query, data)
	if err != nil {
		return nil, fmt.Errorf("named query slice: %w", err)
	}

	q = sqlx.Rebind(sqlx.DOLLAR, q)

	rows, err := db.QueryxContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("query slice: %w", err)
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		var item T
		if err := rows.StructScan(&item); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// QuerySlice executes a query and scans all results into a slice.
func QuerySlice[T any](ctx context.Context, log *logger.Logger, db sqlx.ExtContext, query string) ([]T, error) {
	rows, err := db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query slice: %w", err)
	}
	defer rows.Close()

	var items []T
	for rows.Next() {
		var item T
		if err := rows.StructScan(&item); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}
