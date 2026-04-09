package activitylogdb

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

func (s *Store) Create(ctx context.Context, l activitylogbus.Log) error {
	const q = `
	INSERT INTO activity_logs
		(log_id, subject_type, subject_id, value, logged_at)
	VALUES
		(:log_id, :subject_type, :subject_id, :value, :logged_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBLog(l)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter activitylogbus.QueryFilter, orderBy order.By, pg page.Page) ([]activitylogbus.Log, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT log_id, subject_type, subject_id, value, logged_at FROM activity_logs WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbLogs, err := sqldb.NamedQuerySlice[logDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusLogs(dbLogs), nil
}

func (s *Store) Count(ctx context.Context, filter activitylogbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM activity_logs WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryStreaks(ctx context.Context, subjectType string, subjectID uuid.UUID) (activitylogbus.StreakInfo, error) {
	type summaryRow struct {
		Total      int        `db:"total"`
		LastLogged *time.Time `db:"last_logged"`
	}

	data := map[string]any{
		"subject_type": subjectType,
		"subject_id":   subjectID,
	}

	const summaryQ = `SELECT COUNT(*) as total, MAX(logged_at) as last_logged FROM activity_logs WHERE subject_type = :subject_type AND subject_id = :subject_id`

	var summary summaryRow
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, summaryQ, data, &summary); err != nil {
		return activitylogbus.StreakInfo{}, fmt.Errorf("namedquerystruct summary: %w", err)
	}

	if summary.Total == 0 {
		return activitylogbus.StreakInfo{}, nil
	}

	const datesQ = `SELECT DISTINCT DATE(logged_at) as log_date FROM activity_logs WHERE subject_type = :subject_type AND subject_id = :subject_id ORDER BY log_date DESC`

	dates, err := sqldb.NamedQuerySlice[logDateRow](ctx, s.log, s.db, datesQ, data)
	if err != nil {
		return activitylogbus.StreakInfo{}, fmt.Errorf("namedqueryslice dates: %w", err)
	}

	current, longest := computeStreaks(dates)

	return activitylogbus.StreakInfo{
		Current:    current,
		Longest:    longest,
		TotalCount: summary.Total,
		LastLogged: summary.LastLogged,
	}, nil
}

func (s *Store) QueryBySubjects(ctx context.Context, filter activitylogbus.QueryBySubjectsFilter) ([]activitylogbus.Log, error) {
	data := map[string]any{
		"subject_type": filter.SubjectType,
		"from":         filter.From,
		"to":           filter.To,
	}

	placeholders := make([]string, len(filter.SubjectIDs))
	for i, id := range filter.SubjectIDs {
		key := fmt.Sprintf("subject_id_%d", i)
		placeholders[i] = ":" + key
		data[key] = id
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT log_id, subject_type, subject_id, value, logged_at FROM activity_logs WHERE subject_type = :subject_type`)
	buf.WriteString(fmt.Sprintf(" AND subject_id IN (%s)", strings.Join(placeholders, ", ")))
	buf.WriteString(" AND logged_at >= :from AND logged_at <= :to ORDER BY logged_at DESC")

	dbLogs, err := sqldb.NamedQuerySlice[logDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusLogs(dbLogs), nil
}

type logDateRow struct {
	LogDate time.Time `db:"log_date"`
}

// computeStreaks calculates current and longest streaks from a slice of distinct dates sorted DESC.
// Current streak = consecutive days ending today or yesterday.
// Longest streak = longest run of consecutive days in history.
func computeStreaks(dates []logDateRow) (current, longest int) {
	if len(dates) == 0 {
		return 0, 0
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)

	// Check if the most recent log is today or yesterday (streak still active).
	first := dates[0].LogDate.UTC().Truncate(24 * time.Hour)
	gapFromToday := int(today.Sub(first).Hours() / 24)
	streakActive := gapFromToday <= 1

	runLen := 1
	longest = 1
	currentStreak := 1

	for i := 1; i < len(dates); i++ {
		prev := dates[i-1].LogDate.UTC().Truncate(24 * time.Hour)
		curr := dates[i].LogDate.UTC().Truncate(24 * time.Hour)
		gap := int(prev.Sub(curr).Hours() / 24)

		if gap == 1 {
			runLen++
			if runLen > longest {
				longest = runLen
			}
			if streakActive {
				currentStreak = runLen
			}
		} else {
			streakActive = false
			runLen = 1
		}
	}

	if streakActive {
		current = currentStreak
	}

	return current, longest
}
