package splitapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/splitbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"created_at": splitbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), splitbus.DefaultOrderBy)
}
