package mid

import (
	"context"
	"net/http"
	"time"

	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

func Logger(log *logger.Logger) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			start := time.Now()

			log.Info(ctx, "request started",
				"method", r.Method,
				"path", r.URL.Path,
				"remoteaddr", r.RemoteAddr,
			)

			resp := handler(ctx, r)

			log.Info(ctx, "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"duration", time.Since(start).String(),
			)

			return resp
		}
	}
}
