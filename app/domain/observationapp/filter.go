package observationapp

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/observationbus"
	"github.com/casebrophy/planner/business/types/observationkind"
)

func parseFilter(r *http.Request) (observationbus.QueryFilter, error) {
	var filter observationbus.QueryFilter

	if v := r.URL.Query().Get("subject_type"); v != "" {
		filter.SubjectType = &v
	}

	if v := r.URL.Query().Get("subject_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return observationbus.QueryFilter{}, err
		}
		filter.SubjectID = &id
	}

	if v := r.URL.Query().Get("kind"); v != "" {
		k, err := observationkind.Parse(v)
		if err != nil {
			return observationbus.QueryFilter{}, err
		}
		filter.Kind = &k
	}

	return filter, nil
}
