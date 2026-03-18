package emaildb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/emailbus"
)

func applyFilter(filter emailbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.FromAddress != nil {
		buf.WriteString(" AND from_address = :filter_from_address")
		data["filter_from_address"] = *filter.FromAddress
	}
	if filter.Subject != nil {
		buf.WriteString(" AND subject ILIKE :filter_subject")
		data["filter_subject"] = "%" + *filter.Subject + "%"
	}
}
