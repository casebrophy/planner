package tagdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
)

type Store struct {
	log *logger.Logger
	db  *sqlx.DB
}

func New(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{
		log: log,
		db:  db,
	}
}

// Create inserts a new tag into the database.
func (s *Store) Create(ctx context.Context, tag tagbus.Tag) error {
	const query = `
		INSERT INTO tags (tag_id, name)
		VALUES (:tag_id, :name)
	`

	dbTag := toDBTag(tag)
	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, dbTag); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}

// Delete removes a tag from the database.
func (s *Store) Delete(ctx context.Context, tag tagbus.Tag) error {
	const query = `
		DELETE FROM tags
		WHERE tag_id = :tag_id
	`

	data := map[string]any{
		"tag_id": tag.ID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, data); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

// Query retrieves tags based on filter, order, and pagination.
func (s *Store) Query(ctx context.Context, filter tagbus.QueryFilter, orderBy order.By, pg page.Page) ([]tagbus.Tag, error) {
	orderByStr, err := orderByClause(orderBy)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	var buf bytes.Buffer
	buf.WriteString(`
		SELECT
			tag_id, name
		FROM tags
		WHERE 1=1
	`)

	data := map[string]any{}
	applyFilter(filter, data, &buf)

	buf.WriteString(" ORDER BY " + orderByStr)
	buf.WriteString(fmt.Sprintf(" LIMIT :limit OFFSET :offset"))

	data["limit"] = pg.RowsPerPage()
	data["offset"] = pg.Offset()

	dbTags, err := sqldb.NamedQuerySlice[tagDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("query: %w", err)
	}

	return toBusTags(dbTags), nil
}

// Count returns the number of tags matching the filter.
func (s *Store) Count(ctx context.Context, filter tagbus.QueryFilter) (int, error) {
	var buf bytes.Buffer
	buf.WriteString(`
		SELECT COUNT(*) as count
		FROM tags
		WHERE 1=1
	`)

	data := map[string]any{}
	applyFilter(filter, data, &buf)

	var countResult struct {
		Count int `db:"count"`
	}

	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &countResult); err != nil {
		return 0, fmt.Errorf("count: %w", err)
	}

	return countResult.Count, nil
}

// AddToTask associates a tag with a task.
func (s *Store) AddToTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	const query = `
		INSERT INTO task_tags (task_id, tag_id)
		VALUES (:task_id, :tag_id)
	`

	data := map[string]any{
		"task_id": taskID,
		"tag_id":  tagID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, data); err != nil {
		return fmt.Errorf("add to task: %w", err)
	}

	return nil
}

// RemoveFromTask removes the association between a tag and a task.
func (s *Store) RemoveFromTask(ctx context.Context, taskID, tagID uuid.UUID) error {
	const query = `
		DELETE FROM task_tags
		WHERE task_id = :task_id AND tag_id = :tag_id
	`

	data := map[string]any{
		"task_id": taskID,
		"tag_id":  tagID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, data); err != nil {
		return fmt.Errorf("remove from task: %w", err)
	}

	return nil
}

// AddToContext associates a tag with a context.
func (s *Store) AddToContext(ctx context.Context, contextID, tagID uuid.UUID) error {
	const query = `
		INSERT INTO context_tags (context_id, tag_id)
		VALUES (:context_id, :tag_id)
	`

	data := map[string]any{
		"context_id": contextID,
		"tag_id":     tagID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, data); err != nil {
		return fmt.Errorf("add to context: %w", err)
	}

	return nil
}

// RemoveFromContext removes the association between a tag and a context.
func (s *Store) RemoveFromContext(ctx context.Context, contextID, tagID uuid.UUID) error {
	const query = `
		DELETE FROM context_tags
		WHERE context_id = :context_id AND tag_id = :tag_id
	`

	data := map[string]any{
		"context_id": contextID,
		"tag_id":     tagID,
	}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, query, data); err != nil {
		return fmt.Errorf("remove from context: %w", err)
	}

	return nil
}

// QueryByTask retrieves all tags associated with a task, ordered by name.
func (s *Store) QueryByTask(ctx context.Context, taskID uuid.UUID) ([]tagbus.Tag, error) {
	const query = `
		SELECT
			t.tag_id, t.name
		FROM tags t
		JOIN task_tags tt ON t.tag_id = tt.tag_id
		WHERE tt.task_id = :task_id
		ORDER BY t.name ASC
	`

	data := map[string]any{
		"task_id": taskID,
	}

	dbTags, err := sqldb.NamedQuerySlice[tagDB](ctx, s.log, s.db, query, data)
	if err != nil {
		return nil, fmt.Errorf("query by task: %w", err)
	}

	return toBusTags(dbTags), nil
}

// QueryByContext retrieves all tags associated with a context, ordered by name.
func (s *Store) QueryByContext(ctx context.Context, contextID uuid.UUID) ([]tagbus.Tag, error) {
	const query = `
		SELECT
			t.tag_id, t.name
		FROM tags t
		JOIN context_tags ct ON t.tag_id = ct.tag_id
		WHERE ct.context_id = :context_id
		ORDER BY t.name ASC
	`

	data := map[string]any{
		"context_id": contextID,
	}

	dbTags, err := sqldb.NamedQuerySlice[tagDB](ctx, s.log, s.db, query, data)
	if err != nil {
		return nil, fmt.Errorf("query by context: %w", err)
	}

	return toBusTags(dbTags), nil
}
