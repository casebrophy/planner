package contextdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	contextbus.OrderByID:        "context_id",
	contextbus.OrderByTitle:     "title",
	contextbus.OrderByStatus:    "status",
	contextbus.OrderByLastEvent: "last_event",
	contextbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
