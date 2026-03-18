package clarificationapp

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/clarificationstatus"
)

func parseFilter(r *http.Request) (clarificationbus.QueryFilter, error) {
	var filter clarificationbus.QueryFilter

	if v := r.URL.Query().Get("status"); v != "" {
		s, err := clarificationstatus.Parse(v)
		if err != nil {
			return clarificationbus.QueryFilter{}, err
		}
		filter.Status = &s
	}

	if v := r.URL.Query().Get("kind"); v != "" {
		k, err := clarificationkind.Parse(v)
		if err != nil {
			return clarificationbus.QueryFilter{}, err
		}
		filter.Kind = &k
	}

	if v := r.URL.Query().Get("subject_type"); v != "" {
		filter.SubjectType = &v
	}

	if v := r.URL.Query().Get("subject_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return clarificationbus.QueryFilter{}, err
		}
		filter.SubjectID = &id
	}

	return filter, nil
}
