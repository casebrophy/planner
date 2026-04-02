package timeblockdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

func applyFilter(filter timeblockbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.TaskID != nil {
		buf.WriteString(" AND task_id = :filter_task_id")
		data["filter_task_id"] = *filter.TaskID
	}
	if filter.DateFrom != nil {
		buf.WriteString(" AND starts_at >= :date_from")
		data["date_from"] = *filter.DateFrom
	}
	if filter.DateTo != nil {
		buf.WriteString(" AND ends_at <= :date_to")
		data["date_to"] = *filter.DateTo
	}
	if filter.Confirmed != nil {
		buf.WriteString(" AND confirmed = :filter_confirmed")
		data["filter_confirmed"] = *filter.Confirmed
	}
}
