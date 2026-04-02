package notedb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	notebus.OrderByCreatedAt: "created_at",
	notebus.OrderByUpdatedAt: "updated_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
