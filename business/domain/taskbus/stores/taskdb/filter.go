package taskdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/taskbus"
)

func applyFilter(filter taskbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ID != nil {
		buf.WriteString(" AND task_id = :id")
		data["id"] = *filter.ID
	}
	if filter.Status != nil {
		buf.WriteString(" AND status = :filter_status")
		data["filter_status"] = filter.Status.String()
	}
	if filter.Priority != nil {
		buf.WriteString(" AND priority = :filter_priority")
		data["filter_priority"] = filter.Priority.String()
	}
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.StartDueDate != nil {
		buf.WriteString(" AND due_date >= :start_due_date")
		data["start_due_date"] = *filter.StartDueDate
	}
	if filter.EndDueDate != nil {
		buf.WriteString(" AND due_date <= :end_due_date")
		data["end_due_date"] = *filter.EndDueDate
	}
}
