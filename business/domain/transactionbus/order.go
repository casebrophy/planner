package transactionbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByDate      = "date"
	OrderByAmount    = "amount"
	OrderByCreatedAt = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByDate, order.DESC)
