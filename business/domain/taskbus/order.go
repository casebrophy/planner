package taskbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByID        = "task_id"
	OrderByTitle     = "title"
	OrderByStatus    = "status"
	OrderByPriority  = "priority"
	OrderByDueDate   = "due_date"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
