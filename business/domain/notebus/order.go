package notebus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByCreatedAt = "created_at"
	OrderByUpdatedAt = "updated_at"
)

var DefaultOrderBy = order.NewBy(OrderByCreatedAt, order.DESC)
