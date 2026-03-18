package contextdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/contextbus"
)

func applyFilter(filter contextbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ID != nil {
		buf.WriteString(" AND context_id = :id")
		data["id"] = *filter.ID
	}
	if filter.Status != nil {
		buf.WriteString(" AND status = :filter_status")
		data["filter_status"] = filter.Status.String()
	}
	if filter.Title != nil {
		buf.WriteString(" AND title ILIKE :filter_title")
		data["filter_title"] = "%" + *filter.Title + "%"
	}
}
