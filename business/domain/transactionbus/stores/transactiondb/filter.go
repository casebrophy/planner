package transactiondb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

func applyFilter(filter transactionbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.ContextID != nil {
		buf.WriteString(" AND context_id = :filter_context_id")
		data["filter_context_id"] = *filter.ContextID
	}
	if filter.Source != nil {
		buf.WriteString(" AND source = :filter_source")
		data["filter_source"] = *filter.Source
	}
	if filter.Reviewed != nil {
		buf.WriteString(" AND reviewed = :filter_reviewed")
		data["filter_reviewed"] = *filter.Reviewed
	}
	if filter.Category != nil {
		buf.WriteString(" AND category = :filter_category")
		data["filter_category"] = *filter.Category
	}
}
