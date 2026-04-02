package activitylogapp

import (
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/activitylogbus"
)

func parseFilter(r *http.Request) (activitylogbus.QueryFilter, error) {
	var filter activitylogbus.QueryFilter

	if v := r.URL.Query().Get("subject_type"); v != "" {
		filter.SubjectType = &v
	}

	if v := r.URL.Query().Get("subject_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return activitylogbus.QueryFilter{}, err
		}
		filter.SubjectID = &id
	}

	if v := r.URL.Query().Get("start_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return activitylogbus.QueryFilter{}, err
		}
		filter.StartDate = &t
	}

	if v := r.URL.Query().Get("end_date"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			return activitylogbus.QueryFilter{}, err
		}
		filter.EndDate = &t
	}

	return filter, nil
}
