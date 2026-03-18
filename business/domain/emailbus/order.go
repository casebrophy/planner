package emailbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByReceivedAt = "received_at"
	OrderBySubject    = "subject"
	OrderByCreatedAt  = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByReceivedAt, order.DESC)
