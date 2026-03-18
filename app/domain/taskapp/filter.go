package taskapp

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/types/taskpriority"
	"github.com/casebrophy/planner/business/types/taskstatus"
)

func parseFilter(r *http.Request) (taskbus.QueryFilter, error) {
	var filter taskbus.QueryFilter

	if v := r.URL.Query().Get("status"); v != "" {
		s, err := taskstatus.Parse(v)
		if err != nil {
			return taskbus.QueryFilter{}, err
		}
		filter.Status = &s
	}

	if v := r.URL.Query().Get("priority"); v != "" {
		p, err := taskpriority.Parse(v)
		if err != nil {
			return taskbus.QueryFilter{}, err
		}
		filter.Priority = &p
	}

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return taskbus.QueryFilter{}, err
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("start_due_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return taskbus.QueryFilter{}, err
		}
		filter.StartDueDate = &t
	}

	if v := r.URL.Query().Get("end_due_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return taskbus.QueryFilter{}, err
		}
		filter.EndDueDate = &t
	}

	return filter, nil
}
