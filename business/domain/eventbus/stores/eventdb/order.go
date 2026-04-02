package eventdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	eventbus.OrderByStartsAt:  "starts_at",
	eventbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
