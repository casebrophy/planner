package eventbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByStartsAt  = "starts_at"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByStartsAt, order.ASC)
