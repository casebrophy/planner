package mid

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

func Panics(log *logger.Logger) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) (resp web.Encoder) {
			defer func() {
				if rec := recover(); rec != nil {
					trace := debug.Stack()
					log.Error(ctx, "PANIC",
						"error", fmt.Sprintf("%v", rec),
						"stack", string(trace),
					)
					resp = errs.Newf(errs.Internal, "internal error")
				}
			}()

			return handler(ctx, r)
		}
	}
}
