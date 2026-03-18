package emaildb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/emailbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	emailbus.OrderByReceivedAt: "received_at",
	emailbus.OrderBySubject:    "subject",
	emailbus.OrderByCreatedAt:  "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
