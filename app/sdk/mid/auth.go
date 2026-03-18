package mid

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/foundation/web"
)

func Auth(apiKey string) web.MidFunc {
	return func(handler web.HandlerFunc) web.HandlerFunc {
		return func(ctx context.Context, r *http.Request) web.Encoder {
			key := r.Header.Get("X-API-Key")
			if key == "" {
				auth := r.Header.Get("Authorization")
				if strings.HasPrefix(auth, "Bearer ") {
					key = strings.TrimPrefix(auth, "Bearer ")
				}
			}

			if key == "" {
				return errs.New(errs.Unauthenticated, fmt.Errorf("missing api key"))
			}

			if subtle.ConstantTimeCompare([]byte(key), []byte(apiKey)) != 1 {
				return errs.New(errs.Unauthenticated, fmt.Errorf("invalid api key"))
			}

			return handler(ctx, r)
		}
	}
}
