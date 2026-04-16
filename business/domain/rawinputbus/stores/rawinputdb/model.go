package rawinputdb

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

type rawInputDB struct {
	ID               uuid.UUID        `db:"raw_input_id"`
	SourceType       string           `db:"source_type"`
	Status           string           `db:"status"`
	RawContent       string           `db:"raw_content"`
	ProcessedAt      *time.Time       `db:"processed_at"`
	Error            *string          `db:"error"`
	RetryCount       int              `db:"retry_count"`
	NextRetryAt      *time.Time       `db:"next_retry_at"`
	MaxRetries       int              `db:"max_retries"`
	Result           *json.RawMessage `db:"result"`
	CreatedAt        time.Time        `db:"created_at"`
	UserCorrection   *string          `db:"user_correction"`
	SourceEntityID   uuid.NullUUID    `db:"source_entity_id"`
	SourceEntityKind sql.NullString   `db:"source_entity_kind"`
	SkipClassify     bool             `db:"skip_classify"`
	ReingestMode     bool             `db:"reingest_mode"`
}

func toDBRawInput(ri rawinputbus.RawInput) rawInputDB {
	var result *json.RawMessage
	if ri.Result != nil {
		result = &ri.Result
	}

	sourceEntityID := uuid.NullUUID{Valid: false}
	if ri.SourceEntityID != nil {
		sourceEntityID = uuid.NullUUID{UUID: *ri.SourceEntityID, Valid: true}
	}

	return rawInputDB{
		ID:               ri.ID,
		SourceType:       ri.SourceType.String(),
		Status:           ri.Status.String(),
		RawContent:       ri.RawContent,
		ProcessedAt:      ri.ProcessedAt,
		Error:            ri.Error,
		RetryCount:       ri.RetryCount,
		NextRetryAt:      ri.NextRetryAt,
		MaxRetries:       ri.MaxRetries,
		Result:           result,
		CreatedAt:        ri.CreatedAt,
		UserCorrection:   ri.UserCorrection,
		SourceEntityID:   sourceEntityID,
		SourceEntityKind: sql.NullString{String: ri.SourceEntityKind, Valid: ri.SourceEntityKind != ""},
		SkipClassify:     ri.SkipClassify,
		ReingestMode:     ri.ReingestMode,
	}
}

func toBusRawInput(ri rawInputDB) rawinputbus.RawInput {
	var result json.RawMessage
	if ri.Result != nil {
		result = *ri.Result
	}

	var sourceEntityID *uuid.UUID
	if ri.SourceEntityID.Valid {
		sourceEntityID = &ri.SourceEntityID.UUID
	}

	return rawinputbus.RawInput{
		ID:               ri.ID,
		SourceType:       rawinputsource.MustParse(ri.SourceType),
		Status:           rawinputstatus.MustParse(ri.Status),
		RawContent:       ri.RawContent,
		ProcessedAt:      ri.ProcessedAt,
		Error:            ri.Error,
		RetryCount:       ri.RetryCount,
		NextRetryAt:      ri.NextRetryAt,
		MaxRetries:       ri.MaxRetries,
		Result:           result,
		CreatedAt:        ri.CreatedAt,
		UserCorrection:   ri.UserCorrection,
		SourceEntityID:   sourceEntityID,
		SourceEntityKind: ri.SourceEntityKind.String,
		SkipClassify:     ri.SkipClassify,
		ReingestMode:     ri.ReingestMode,
	}
}

func toBusRawInputs(ris []rawInputDB) []rawinputbus.RawInput {
	items := make([]rawinputbus.RawInput, len(ris))
	for i, ri := range ris {
		items[i] = toBusRawInput(ri)
	}
	return items
}
