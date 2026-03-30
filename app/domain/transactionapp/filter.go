package transactionapp

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/transactionbus"
)

func parseFilter(r *http.Request) (transactionbus.QueryFilter, error) {
	var filter transactionbus.QueryFilter

	if v := r.URL.Query().Get("context_id"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return transactionbus.QueryFilter{}, fmt.Errorf("invalid context_id: %w", err)
		}
		filter.ContextID = &id
	}

	if v := r.URL.Query().Get("source"); v != "" {
		filter.Source = &v
	}

	if v := r.URL.Query().Get("reviewed"); v != "" {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return transactionbus.QueryFilter{}, fmt.Errorf("invalid reviewed: %w", err)
		}
		filter.Reviewed = &b
	}

	if v := r.URL.Query().Get("category"); v != "" {
		filter.Category = &v
	}

	return filter, nil
}
