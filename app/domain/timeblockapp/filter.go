package timeblockapp

import (
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/timeblockbus"
)

func parseFilter(r *http.Request) (timeblockbus.QueryFilter, error) {
	var filter timeblockbus.QueryFilter

	if v := r.URL.Query().Get("task_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.TaskID = &id
	}

	if v := r.URL.Query().Get("date_from"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.DateFrom = &t
	}

	if v := r.URL.Query().Get("date_to"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.DateTo = &t
	}

	if v := r.URL.Query().Get("confirmed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return timeblockbus.QueryFilter{}, err
		}
		filter.Confirmed = &b
	}

	return filter, nil
}
