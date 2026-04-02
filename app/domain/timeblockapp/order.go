package timeblockapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"starts_at":  timeblockbus.OrderByStartsAt,
	"created_at": timeblockbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), timeblockbus.DefaultOrderBy)
}
