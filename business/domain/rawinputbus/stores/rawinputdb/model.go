package rawinputdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type rawInputDB struct {
	ID          uuid.UUID  `db:"raw_input_id"`
	SourceType  string     `db:"source_type"`
	Status      string     `db:"status"`
	RawContent  string     `db:"raw_content"`
	ProcessedAt *time.Time `db:"processed_at"`
	Error       *string    `db:"error"`
	CreatedAt   time.Time  `db:"created_at"`
}

func toDBRawInput(ri rawinputbus.RawInput) rawInputDB {
	return rawInputDB{
		ID:          ri.ID,
		SourceType:  ri.SourceType.String(),
		Status:      ri.Status.String(),
		RawContent:  ri.RawContent,
		ProcessedAt: ri.ProcessedAt,
		Error:       ri.Error,
		CreatedAt:   ri.CreatedAt,
	}
}

func toBusRawInput(ri rawInputDB) rawinputbus.RawInput {
	return rawinputbus.RawInput{
		ID:          ri.ID,
		SourceType:  rawinputsource.MustParse(ri.SourceType),
		Status:      rawinputstatus.MustParse(ri.Status),
		RawContent:  ri.RawContent,
		ProcessedAt: ri.ProcessedAt,
		Error:       ri.Error,
		CreatedAt:   ri.CreatedAt,
	}
}

func toBusRawInputs(ris []rawInputDB) []rawinputbus.RawInput {
	items := make([]rawinputbus.RawInput, len(ris))
	for i, ri := range ris {
		items[i] = toBusRawInput(ri)
	}
	return items
}
