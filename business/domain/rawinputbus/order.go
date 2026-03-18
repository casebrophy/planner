package rawinputbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByCreatedAt = "created_at"
	OrderByStatus    = "status"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
