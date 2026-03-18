package emailapp

import (
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/emailbus"
)

func parseFilter(r *http.Request) (emailbus.QueryFilter, error) {
	var filter emailbus.QueryFilter

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return emailbus.QueryFilter{}, err
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("from_address"); v != "" {
		filter.FromAddress = &v
	}

	if v := r.URL.Query().Get("subject"); v != "" {
		filter.Subject = &v
	}

	return filter, nil
}
