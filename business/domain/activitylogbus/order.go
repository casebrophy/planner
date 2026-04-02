package activitylogbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByLoggedAt = "logged_at"
)

var DefaultOrderBy = order.NewBy(OrderByLoggedAt, order.DESC)
