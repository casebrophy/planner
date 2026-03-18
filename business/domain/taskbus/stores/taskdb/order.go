package taskdb

import (
	"fmt"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/order"
)

var orderByFields = map[string]string{
	taskbus.OrderByID:        "task_id",
	taskbus.OrderByTitle:     "title",
	taskbus.OrderByStatus:    "status",
	taskbus.OrderByPriority:  "priority",
	taskbus.OrderByDueDate:   "due_date",
	taskbus.OrderByCreatedAt: "created_at",
}

func orderByClause(ob order.By) (string, error) {
	col, ok := orderByFields[ob.Field]
	if !ok {
		return "", fmt.Errorf("unknown order field %q", ob.Field)
	}
	return col + " " + ob.Direction, nil
}
