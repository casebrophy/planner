package dailyplandb

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/dailyplanbus"
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

func (s *Store) CreatePlan(ctx context.Context, plan dailyplanbus.DailyPlan) error {
	const q = `
	INSERT INTO daily_plans
		(plan_id, plan_date, generation, model_used, prompt_hash, created_at)
	VALUES
		(:plan_id, :plan_date, :generation, :model_used, :prompt_hash, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBPlan(plan)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) CreateItem(ctx context.Context, item dailyplanbus.DailyPlanItem) error {
	const q = `
	INSERT INTO daily_plan_items
		(item_id, plan_id, task_id, position, group_name, group_position, ai_duration_min, ai_priority_reason, status, created_at)
	VALUES
		(:item_id, :plan_id, :task_id, :position, :group_name, :group_position, :ai_duration_min, :ai_priority_reason, :status, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBItem(item)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) UpdateItem(ctx context.Context, item dailyplanbus.DailyPlanItem) error {
	const q = `
	UPDATE daily_plan_items SET
		user_position = :user_position,
		user_duration_min = :user_duration_min,
		status = :status,
		dismiss_reason = :dismiss_reason,
		dismiss_note = :dismiss_note,
		completed_at = :completed_at
	WHERE
		item_id = :item_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBItem(item)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}

func (s *Store) QueryPlanByDate(ctx context.Context, date time.Time) (dailyplanbus.DailyPlan, error) {
	data := struct {
		PlanDate time.Time `db:"plan_date"`
	}{
		PlanDate: date,
	}

	const q = `
	SELECT plan_id, plan_date, generation, model_used, prompt_hash, created_at
	FROM daily_plans
	WHERE plan_date = :plan_date
	ORDER BY generation DESC
	LIMIT 1`

	var p dailyPlanDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &p); err != nil {
		return dailyplanbus.DailyPlan{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusPlan(p), nil
}

func (s *Store) QueryItemsByPlan(ctx context.Context, planID uuid.UUID) ([]dailyplanbus.DailyPlanItem, error) {
	data := struct {
		PlanID uuid.UUID `db:"plan_id"`
	}{
		PlanID: planID,
	}

	const q = `
	SELECT item_id, plan_id, task_id, position, group_name, group_position, ai_duration_min, ai_priority_reason, user_position, user_duration_min, status, dismiss_reason, dismiss_note, completed_at, created_at
	FROM daily_plan_items
	WHERE plan_id = :plan_id
	ORDER BY group_position, position`

	items, err := sqldb.NamedQuerySlice[dailyPlanItemDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	return toBusItems(items), nil
}

func (s *Store) QueryItemByID(ctx context.Context, itemID uuid.UUID) (dailyplanbus.DailyPlanItem, error) {
	data := struct {
		ItemID uuid.UUID `db:"item_id"`
	}{
		ItemID: itemID,
	}

	const q = `
	SELECT item_id, plan_id, task_id, position, group_name, group_position, ai_duration_min, ai_priority_reason, user_position, user_duration_min, status, dismiss_reason, dismiss_note, completed_at, created_at
	FROM daily_plan_items
	WHERE item_id = :item_id`

	var i dailyPlanItemDB
	if err := sqldb.NamedQueryStruct(ctx, s.log, s.db, q, data, &i); err != nil {
		return dailyplanbus.DailyPlanItem{}, fmt.Errorf("namedquerystruct: %w", err)
	}

	return toBusItem(i), nil
}

func (s *Store) DeleteItemsByPlan(ctx context.Context, planID uuid.UUID) error {
	data := struct {
		PlanID uuid.UUID `db:"plan_id"`
	}{
		PlanID: planID,
	}

	const q = `DELETE FROM daily_plan_items WHERE plan_id = :plan_id`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}

	return nil
}
