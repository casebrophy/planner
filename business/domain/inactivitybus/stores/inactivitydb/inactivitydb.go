package inactivitydb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/inactivitybus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
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

// QueryOverlappingContexts returns pairs of active contexts sharing 2+ tags.
func (s *Store) QueryOverlappingContexts(ctx context.Context) ([]inactivitybus.OverlapPair, error) {
	const q = `
		SELECT
			ct1.context_id AS context_id_1,
			c1.title AS title_1,
			ct2.context_id AS context_id_2,
			c2.title AS title_2,
			COUNT(*) AS shared_tags
		FROM context_tags ct1
		JOIN context_tags ct2 ON ct1.tag_id = ct2.tag_id AND ct1.context_id < ct2.context_id
		JOIN contexts c1 ON c1.context_id = ct1.context_id AND c1.status = 'active'
		JOIN contexts c2 ON c2.context_id = ct2.context_id AND c2.status = 'active'
		GROUP BY ct1.context_id, c1.title, ct2.context_id, c2.title
		HAVING COUNT(*) >= 2
		ORDER BY shared_tags DESC
		LIMIT 10`

	type dbOverlapPair struct {
		ContextID1 uuid.UUID `db:"context_id_1"`
		Title1     string    `db:"title_1"`
		ContextID2 uuid.UUID `db:"context_id_2"`
		Title2     string    `db:"title_2"`
		SharedTags int       `db:"shared_tags"`
	}

	rows, err := sqldb.NamedQuerySlice[dbOverlapPair](ctx, s.log, s.db, q, struct{}{})
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice overlapping contexts: %w", err)
	}

	pairs := make([]inactivitybus.OverlapPair, len(rows))
	for i, r := range rows {
		pairs[i] = inactivitybus.OverlapPair{
			ContextID1: r.ContextID1,
			Title1:     r.Title1,
			ContextID2: r.ContextID2,
			Title2:     r.Title2,
			SharedTags: r.SharedTags,
		}
	}

	return pairs, nil
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
