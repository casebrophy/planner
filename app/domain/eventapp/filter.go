package eventapp

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/eventbus"
)

func parseFilter(r *http.Request) (eventbus.QueryFilter, error) {
	var filter eventbus.QueryFilter

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return eventbus.QueryFilter{}, err
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("date_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return eventbus.QueryFilter{}, err
		}
		filter.DateFrom = &t
	}

	if v := r.URL.Query().Get("date_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return eventbus.QueryFilter{}, err
		}
		filter.DateTo = &t
	}

	return filter, nil
}
