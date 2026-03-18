package clarificationbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByPriorityScore = "priority_score"
	OrderByCreatedAt     = "created_at"
)

var DefaultOrderBy = order.NewBy(OrderByPriorityScore, order.DESC)
