package noteapp

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/notebus"
)

func parseFilter(r *http.Request) (notebus.QueryFilter, error) {
	var filter notebus.QueryFilter

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return notebus.QueryFilter{}, err
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("task_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return notebus.QueryFilter{}, err
		}
		filter.TaskID = &id
	}

	if v := r.URL.Query().Get("source"); v != "" {
		filter.Source = &v
	}

	if v := r.URL.Query().Get("search"); v != "" {
		filter.Search = &v
	}

	return filter, nil
}
