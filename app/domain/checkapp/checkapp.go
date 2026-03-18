package checkapp

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/sqldb"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	db *sqlx.DB
}

type status struct {
	Status string `json:"status"`
}

func (s status) Encode() ([]byte, string, error) {
	data, err := json.Marshal(s)
	return data, "application/json", err
}

func (a *app) readiness(ctx context.Context, r *http.Request) web.Encoder {
	if err := sqldb.StatusCheck(ctx, a.db); err != nil {
		return errs.Newf(errs.Internal, "database not ready: %s", err)
	}
	return status{Status: "ok"}
}

func (a *app) liveness(ctx context.Context, r *http.Request) web.Encoder {
	return status{Status: "ok"}
}
