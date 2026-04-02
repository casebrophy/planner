package timeblockdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	timeblockbus.OrderByStartsAt:  "starts_at",
	timeblockbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
