package clarificationapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"priority_score": clarificationbus.OrderByPriorityScore,
	"created_at":     clarificationbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), clarificationbus.DefaultOrderBy)
}
