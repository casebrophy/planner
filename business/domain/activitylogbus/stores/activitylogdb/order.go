package activitylogdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	activitylogbus.OrderByLoggedAt: "logged_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
