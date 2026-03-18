package mid

import (
	"context"
	"net/http"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

func Errors(log *logger.Logger) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			resp := handler(ctx, r)

			if e, ok := resp.(*errs.Error); ok {
				log.Error(ctx, "request error",
					"message", e.Message,
					"code", e.Code,
					"func", e.FuncName,
				)
			}

			return resp
		}
	}
}
