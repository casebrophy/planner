package tagapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"id":   tagbus.OrderByID,
	"name": tagbus.OrderByName,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), tagbus.DefaultOrderBy)
}
