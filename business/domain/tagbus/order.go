package tagbus

import "github.com/casebrophy/planner/business/sdk/order"

const (
	OrderByID   = "tag_id"
	OrderByName = "name"
)

var DefaultOrderBy = order.NewBy(OrderByName, order.ASC)
