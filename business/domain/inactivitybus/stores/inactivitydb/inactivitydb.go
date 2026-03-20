package inactivitydb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
)

// Store manages inactivity detection queries.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore creates a new inactivity store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

// dbStaleItem is the database representation of a stale item.
type dbStaleItem struct {
	SubjectType   string    `db:"subject_type"`
	SubjectID     uuid.UUID `db:"subject_id"`
	Title         string    `db:"title"`
	Priority      string    `db:"priority"`
	LastUpdated   time.Time `db:"last_updated"`
	ThresholdDays float64   `db:"threshold_days"`
}

func toBusStaleItem(db dbStaleItem) inactivitybus.StaleItem {
	return inactivitybus.StaleItem{
		SubjectType:   db.SubjectType,
		SubjectID:     db.SubjectID,
		Title:         db.Title,
		Priority:      db.Priority,
		LastUpdated:   db.LastUpdated,
		ThresholdDays: db.ThresholdDays,
	}
}

func toBusStaleItems(dbs []dbStaleItem) []inactivitybus.StaleItem {
	items := make([]inactivitybus.StaleItem, len(dbs))
	for i, db := range dbs {
		items[i] = toBusStaleItem(db)
	}
	return items
}

// QueryStaleTasks returns tasks exceeding priority-based inactivity thresholds.
// Thresholds: urgent=1d, high=2d, medium=5d, low=14d.
// Only includes tasks in todo or in_progress status.
func (s *Store) QueryStaleTasks(ctx context.Context) ([]inactivitybus.StaleItem, error) {
	const q = `
	SELECT
		'task' AS subject_type,
		task_id AS subject_id,
		title,
		priority,
		COALESCE(last_thread_at, updated_at) AS last_updated,
		CASE priority
			WHEN 'urgent' THEN 1
			WHEN 'high' THEN 2
			WHEN 'medium' THEN 5
			WHEN 'low' THEN 14
			ELSE 7
		END AS threshold_days
	FROM tasks
	WHERE status IN ('todo', 'in_progress')
		AND COALESCE(last_thread_at, updated_at) < NOW() - INTERVAL '1 day' *
			CASE priority
				WHEN 'urgent' THEN 1
				WHEN 'high' THEN 2
				WHEN 'medium' THEN 5
				WHEN 'low' THEN 14
				ELSE 7
			END`

	data := struct{}{}

	rows, err := sqldb.NamedQuerySlice[dbStaleItem](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice stale tasks: %w", err)
	}

	return toBusStaleItems(rows), nil
}

// QueryStaleContexts returns active contexts exceeding the 7-day default
// inactivity threshold.
func (s *Store) QueryStaleContexts(ctx context.Context) ([]inactivitybus.StaleItem, error) {
	const q = `
	SELECT
		'context' AS subject_type,
		context_id AS subject_id,
		title,
		'medium' AS priority,
		COALESCE(last_event, last_thread_at, updated_at) AS last_updated,
		7 AS threshold_days
	FROM contexts
	WHERE status = 'active'
		AND COALESCE(last_event, last_thread_at, updated_at) < NOW() - INTERVAL '7 days'`

	data := struct{}{}

	rows, err := sqldb.NamedQuerySlice[dbStaleItem](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice stale contexts: %w", err)
	}

	return toBusStaleItems(rows), nil
}
