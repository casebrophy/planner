package rawinputapp

import (
	"net/http"

	"github.com/casebrophy/planner/business/domain/rawinputbus"
	"github.com/casebrophy/planner/business/types/rawinputsource"
	"github.com/casebrophy/planner/business/types/rawinputstatus"
)

func parseFilter(r *http.Request) (rawinputbus.QueryFilter, error) {
	var filter rawinputbus.QueryFilter

	if v := r.URL.Query().Get("status"); v != "" {
		s, err := rawinputstatus.Parse(v)
		if err != nil {
			return rawinputbus.QueryFilter{}, err
		}
		filter.Status = &s
	}

	if v := r.URL.Query().Get("source_type"); v != "" {
		s, err := rawinputsource.Parse(v)
		if err != nil {
			return rawinputbus.QueryFilter{}, err
		}
		filter.SourceType = &s
	}

	return filter, nil
}
