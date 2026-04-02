package dailyplanbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByGroupPosition = "group_position"
	OrderByPosition      = "position"
	OrderByCreatedAt     = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByGroupPosition, order.ASC)
