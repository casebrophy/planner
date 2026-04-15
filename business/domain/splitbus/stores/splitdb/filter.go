package splitdb

import (
	"bytes"

	"github.com/casebrophy/planner/business/domain/splitbus"
)

func applyFilter(filter splitbus.QueryFilter, data map[string]any, buf *bytes.Buffer) {
	if filter.TransactionID != nil {
		buf.WriteString(" AND transaction_id = :filter_transaction_id")
		data["filter_transaction_id"] = *filter.TransactionID
	}
}
