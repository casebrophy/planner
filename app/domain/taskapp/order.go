package taskapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	"id":         taskbus.OrderByID,
	"title":      taskbus.OrderByTitle,
	"status":     taskbus.OrderByStatus,
	"priority":   taskbus.OrderByPriority,
	"due_date":   taskbus.OrderByDueDate,
	"created_at": taskbus.OrderByCreatedAt,
}

func parseOrder(r *http.Request) (order.By, error) {
	return order.Parse(orderByFields, r.URL.Query().Get("orderBy"), taskbus.DefaultOrderBy)
}
