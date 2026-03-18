package tagdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/tagbus"
)

func applyFilter(filter tagbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ID != nil {
		buf.WriteString(" AND tag_id = :id")
		data["id"] = *filter.ID
	}
	if filter.Name != nil {
		buf.WriteString(" AND name ILIKE :filter_name")
		data["filter_name"] = "%" + *filter.Name + "%"
	}
}
