package rawinputapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"created_at": rawinputbus.OrderByCreatedAt,
	"status":     rawinputbus.OrderByStatus,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), rawinputbus.DefaultOrderBy)
}
