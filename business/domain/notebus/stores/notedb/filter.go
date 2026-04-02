package notedb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/notebus"
)

func applyFilter(filter notebus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.Source != nil {
		buf.WriteString(" AND source = :filter_source")
		data["filter_source"] = *filter.Source
	}
	if filter.Search != nil {
		buf.WriteString(" AND content ILIKE :search")
		data["search"] = "%" + *filter.Search + "%"
	}
}
