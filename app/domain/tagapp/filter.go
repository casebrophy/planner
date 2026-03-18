package tagapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/tagbus"
)

func parseFilter(r *http.Request) (tagbus.QueryFilter, error) {
	var filter tagbus.QueryFilter

	if v := r.URL.Query().Get("name"); v != "" {
		filter.Name = &v
	}

	return filter, nil
}
