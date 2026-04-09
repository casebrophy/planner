package taskdb

import (
	"bytes"
	"fmt"
	"strings"

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
	if len(filter.ExcludeStatuses) > 0 {
		placeholders := make([]string, len(filter.ExcludeStatuses))
		for i, s := range filter.ExcludeStatuses {
			key := fmt.Sprintf("excl_status_%d", i)
			placeholders[i] = ":" + key
			data[key] = s.String()
		}
		buf.WriteString(fmt.Sprintf(" AND status NOT IN (%s)", strings.Join(placeholders, ", ")))
	}
	if filter.HasRecurrence != nil {
		if *filter.HasRecurrence {
			buf.WriteString(" AND recurrence_rule IS NOT NULL AND recurrence_rule != ''")
		} else {
			buf.WriteString(" AND (recurrence_rule IS NULL OR recurrence_rule = '')")
		}
	}
}
