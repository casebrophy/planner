package eventdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/eventbus"
)

func applyFilter(filter eventbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.DateFrom != nil {
		buf.WriteString(" AND starts_at >= :date_from")
		data["date_from"] = *filter.DateFrom
	}
	if filter.DateTo != nil {
		buf.WriteString(" AND ends_at <= :date_to")
		data["date_to"] = *filter.DateTo
	}
}
