package contextbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByID        = "context_id"
	OrderByTitle     = "title"
	OrderByStatus    = "status"
	OrderByLastEvent = "last_event"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByLastEvent, order.DESC)
