package contextapp

import (
	"fmt"
	"net/http"

	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/types/contextkind"
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

	if v := r.URL.Query().Get("kind"); v != "" {
		kind, err := contextkind.Parse(v)
		if err != nil {
			return contextbus.QueryFilter{}, fmt.Errorf("invalid kind: %w", err)
		}
		filter.Kind = &kind
	}

	if v := r.URL.Query().Get("title"); v != "" {
		filter.Title = &v
	}

	return filter, nil
}
