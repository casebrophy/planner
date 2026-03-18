package observationbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
