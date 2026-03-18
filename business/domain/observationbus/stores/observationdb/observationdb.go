package observationdb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/sdk/order"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/sqldb"
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

func (s *Store) Create(ctx context.Context, obs observationbus.Observation) error {
	const q = `
	INSERT INTO outcome_observations
		(observation_id, subject_type, subject_id, kind, data, source, confidence, weight, created_at)
	VALUES
		(:observation_id, :subject_type, :subject_id, :kind, :data, :source, :confidence, :weight, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBObservation(obs)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter observationbus.QueryFilter, orderBy order.By, pg page.Page) ([]observationbus.Observation, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT observation_id, subject_type, subject_id, kind, data, source, confidence, weight, created_at FROM outcome_observations WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbObs, err := sqldb.NamedQuerySlice[observationDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusObservations(dbObs), nil
}

func (s *Store) Count(ctx context.Context, filter observationbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM outcome_observations WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}
