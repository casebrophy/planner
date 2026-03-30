package transactionapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/transactionbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"date":       transactionbus.OrderByDate,
	"amount":     transactionbus.OrderByAmount,
	"created_at": transactionbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), transactionbus.DefaultOrderBy)
}
