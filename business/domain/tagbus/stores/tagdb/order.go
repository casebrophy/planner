package tagdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/tagbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	tagbus.OrderByID:   "tag_id",
	tagbus.OrderByName: "name",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
