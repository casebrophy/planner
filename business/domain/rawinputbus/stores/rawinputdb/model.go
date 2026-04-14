package rawinputdb

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type rawInputDB struct {
	ID              uuid.UUID        `db:"raw_input_id"`
	SourceType      string           `db:"source_type"`
	Status          string           `db:"status"`
	RawContent      string           `db:"raw_content"`
	ProcessedAt     *time.Time       `db:"processed_at"`
	Error           *string          `db:"error"`
	RetryCount      int              `db:"retry_count"`
	NextRetryAt     *time.Time       `db:"next_retry_at"`
	MaxRetries      int              `db:"max_retries"`
	Result          *json.RawMessage `db:"result"`
	CreatedAt       time.Time        `db:"created_at"`
	UserCorrection  *string          `db:"user_correction"`
}

func toDBRawInput(ri rawinputbus.RawInput) rawInputDB {
	var result *json.RawMessage
	if ri.Result != nil {
		result = &ri.Result
	}

	return rawInputDB{
		ID:              ri.ID,
		SourceType:      ri.SourceType.String(),
		Status:          ri.Status.String(),
		RawContent:      ri.RawContent,
		ProcessedAt:     ri.ProcessedAt,
		Error:           ri.Error,
		RetryCount:      ri.RetryCount,
		NextRetryAt:     ri.NextRetryAt,
		MaxRetries:      ri.MaxRetries,
		Result:          result,
		CreatedAt:       ri.CreatedAt,
		UserCorrection:  ri.UserCorrection,
	}
}

func toBusRawInput(ri rawInputDB) rawinputbus.RawInput {
	var result json.RawMessage
	if ri.Result != nil {
		result = *ri.Result
	}

	return rawinputbus.RawInput{
		ID:              ri.ID,
		SourceType:      rawinputsource.MustParse(ri.SourceType),
		Status:          rawinputstatus.MustParse(ri.Status),
		RawContent:      ri.RawContent,
		ProcessedAt:     ri.ProcessedAt,
		Error:           ri.Error,
		RetryCount:      ri.RetryCount,
		NextRetryAt:     ri.NextRetryAt,
		MaxRetries:      ri.MaxRetries,
		Result:          result,
		CreatedAt:       ri.CreatedAt,
		UserCorrection:  ri.UserCorrection,
	}
}

func toBusRawInputs(ris []rawInputDB) []rawinputbus.RawInput {
	items := make([]rawinputbus.RawInput, len(ris))
	for i, ri := range ris {
		items[i] = toBusRawInput(ri)
	}
	return items
}
