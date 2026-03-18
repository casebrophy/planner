package contextapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"id":          contextbus.OrderByID,
	"title":       contextbus.OrderByTitle,
	"status":      contextbus.OrderByStatus,
	"last_event":  contextbus.OrderByLastEvent,
	"created_at":  contextbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), contextbus.DefaultOrderBy)
}
