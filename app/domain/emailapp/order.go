package emailapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"received_at": emailbus.OrderByReceivedAt,
	"subject":     emailbus.OrderBySubject,
	"created_at":  emailbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), emailbus.DefaultOrderBy)
}
