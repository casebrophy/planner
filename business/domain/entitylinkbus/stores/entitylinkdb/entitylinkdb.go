package entitylinkdb

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

// Store implements entitylinkbus.Storer against PostgreSQL.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs an entitylinkdb.Store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{log: log, db: db}
}

func (s *Store) Create(ctx context.Context, link entitylinkbus.EntityLink) error {
	const q = `
	INSERT INTO entity_links (link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at)
	VALUES (:link_id, :source_type, :source_id, :target_type, :target_id, :confidence, :kind, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEntityLink(link)); err != nil {
		if errors.Is(err, sqldb.ErrDBDuplicatedEntry) {
			return fmt.Errorf("entity link already exists: %w", sqldb.ErrDBDuplicatedEntry)
		}
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (entitylinkbus.EntityLink, error) {
	const q = `
	SELECT link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at
	FROM entity_links
	WHERE link_id = :link_id`

	data := struct {
		ID uuid.UUID `db:"link_id"`
	}{ID: id}

	var el entityLinkDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &el); err != nil {
		return entitylinkbus.EntityLink{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusEntityLink(el), nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM entity_links WHERE link_id = :link_id`

	data := struct {
		ID uuid.UUID `db:"link_id"`
	}{ID: id}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}

func (s *Store) QueryBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]entitylinkbus.EntityLink, error) {
	const q = `
	SELECT link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at
	FROM entity_links
	WHERE source_type = :source_type AND source_id = :source_id
	ORDER BY created_at DESC`

	data := map[string]any{
		"source_type": sourceType,
		"source_id":   sourceID,
	}

	rows, err := sqldb.NamedQuerySlice[entityLinkDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	links := make([]entitylinkbus.EntityLink, len(rows))
	for i, r := range rows {
		links[i] = toBusEntityLink(r)
	}
	return links, nil
}

func (s *Store) QueryByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]entitylinkbus.EntityLink, error) {
	const q = `
	SELECT link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at
	FROM entity_links
	WHERE target_type = :target_type AND target_id = :target_id
	ORDER BY created_at DESC`

	data := map[string]any{
		"target_type": targetType,
		"target_id":   targetID,
	}

	rows, err := sqldb.NamedQuerySlice[entityLinkDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	links := make([]entitylinkbus.EntityLink, len(rows))
	for i, r := range rows {
		links[i] = toBusEntityLink(r)
	}
	return links, nil
}
