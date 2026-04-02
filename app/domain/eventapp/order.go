package eventapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"starts_at":  eventbus.OrderByStartsAt,
	"created_at": eventbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), eventbus.DefaultOrderBy)
}
