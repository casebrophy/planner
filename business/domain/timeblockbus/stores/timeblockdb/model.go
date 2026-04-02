package timeblockdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

type timeBlockDB struct {
	ID        uuid.UUID `db:"block_id"`
	TaskID    uuid.UUID `db:"task_id"`
	StartsAt  time.Time `db:"starts_at"`
	EndsAt    time.Time `db:"ends_at"`
	Confirmed bool      `db:"confirmed"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func toDBTimeBlock(tb timeblockbus.TimeBlock) timeBlockDB {
	return timeBlockDB{
		ID:        tb.ID,
		TaskID:    tb.TaskID,
		StartsAt:  tb.StartsAt,
		EndsAt:    tb.EndsAt,
		Confirmed: tb.Confirmed,
		CreatedAt: tb.CreatedAt,
		UpdatedAt: tb.UpdatedAt,
	}
}

func toBusTimeBlock(tb timeBlockDB) timeblockbus.TimeBlock {
	return timeblockbus.TimeBlock{
		ID:        tb.ID,
		TaskID:    tb.TaskID,
		StartsAt:  tb.StartsAt,
		EndsAt:    tb.EndsAt,
		Confirmed: tb.Confirmed,
		CreatedAt: tb.CreatedAt,
		UpdatedAt: tb.UpdatedAt,
	}
}

func toBusTimeBlocks(tbs []timeBlockDB) []timeblockbus.TimeBlock {
	blocks := make([]timeblockbus.TimeBlock, len(tbs))
	for i, tb := range tbs {
		blocks[i] = toBusTimeBlock(tb)
	}
	return blocks
}
