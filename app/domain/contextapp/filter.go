package contextapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/contextbus"
)

func parseFilter(r *http.Request) (contextbus.QueryFilter, error) {
	var filter contextbus.QueryFilter

	if v := r.URL.Query().Get("status"); v != "" {
		s, err := contextbus.Parse(v)
		if err != nil {
			return contextbus.QueryFilter{}, err
		}
		filter.Status = &s
	}

	if v := r.URL.Query().Get("title"); v != "" {
		filter.Title = &v
	}

	return filter, nil
}
