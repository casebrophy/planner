package emaildb

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/emailbus"
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

func (s *Store) Create(ctx context.Context, e emailbus.Email) error {
	const q = `
	INSERT INTO emails
		(email_id, raw_input_id, message_id, from_address, from_name, to_address, subject, body_text, body_html, received_at, context_id, created_at)
	VALUES
		(:email_id, :raw_input_id, :message_id, :from_address, :from_name, :to_address, :subject, :body_text, :body_html, :received_at, :context_id, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEmail(e)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Update(ctx context.Context, e emailbus.Email) error {
	const q = `
	UPDATE emails SET
		context_id = :context_id
	WHERE
		email_id = :email_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEmail(e)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Delete(ctx context.Context, e emailbus.Email) error {
	data := struct {
		ID uuid.UUID `db:"email_id"`
	}{
		ID: e.ID,
	}

	const q = `DELETE FROM emails WHERE email_id = :email_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) Query(ctx context.Context, filter emailbus.QueryFilter, orderBy order.By, pg page.Page) ([]emailbus.Email, error) {
	data := map[string]any{
		"offset":        pg.Offset(),
		"rows_per_page": pg.RowsPerPage(),
	}

	var buf bytes.Buffer
	buf.WriteString(`SELECT email_id, raw_input_id, message_id, from_address, from_name, to_address, subject, body_text, body_html, received_at, context_id, created_at FROM emails WHERE 1=1`)

	applyFilter(filter, data, &buf)

	orderClause, err := orderByClause(orderBy)
	if err != nil {
		return nil, err
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY %s OFFSET :offset ROWS FETCH NEXT :rows_per_page ROWS ONLY", orderClause))

	dbItems, err := sqldb.NamedQuerySlice[emailDB](ctx, s.log, s.db, buf.String(), data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusEmails(dbItems), nil
}

func (s *Store) Count(ctx context.Context, filter emailbus.QueryFilter) (int, error) {
	data := map[string]any{}

	var buf bytes.Buffer
	buf.WriteString(`SELECT COUNT(*) FROM emails WHERE 1=1`)

	applyFilter(filter, data, &buf)

	var count struct {
		Count int `db:"count"`
	}
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, buf.String(), data, &count); err != nil {
		return 0, fmt.Errorf("namedquerystruct: %w", err)
	}

	return count.Count, nil
}

func (s *Store) QueryByID(ctx context.Context, id uuid.UUID) (emailbus.Email, error) {
	data := struct {
		ID uuid.UUID `db:"email_id"`
	}{
		ID: id,
	}

	const q = `SELECT email_id, raw_input_id, message_id, from_address, from_name, to_address, subject, body_text, body_html, received_at, context_id, created_at FROM emails WHERE email_id = :email_id`

	var e emailDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &e); err != nil {
		return emailbus.Email{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusEmail(e), nil
}

func (s *Store) QueryByMessageID(ctx context.Context, messageID string) (emailbus.Email, error) {
	data := struct {
		MessageID string `db:"message_id"`
	}{
		MessageID: messageID,
	}

	const q = `SELECT email_id, raw_input_id, message_id, from_address, from_name, to_address, subject, body_text, body_html, received_at, context_id, created_at FROM emails WHERE message_id = :message_id`

	var e emailDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &e); err != nil {
		return emailbus.Email{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusEmail(e), nil
}
