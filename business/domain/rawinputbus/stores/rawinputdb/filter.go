package rawinputdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
)

func applyFilter(filter rawinputbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.Status != nil {
		buf.WriteString(" AND status = :filter_status")
		data["filter_status"] = filter.Status.String()
	}
	if filter.SourceType != nil {
		buf.WriteString(" AND source_type = :filter_source_type")
		data["filter_source_type"] = filter.SourceType.String()
	}
}
