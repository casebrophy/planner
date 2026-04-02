package noteapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"created_at": notebus.OrderByCreatedAt,
	"updated_at": notebus.OrderByUpdatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), notebus.DefaultOrderBy)
}
