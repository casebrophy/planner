package activitylogapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"logged_at": activitylogbus.OrderByLoggedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), activitylogbus.DefaultOrderBy)
}
