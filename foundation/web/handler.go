package web

import (
	"context"
	"net/http"
)

// HandlerFunc is the signature handlers implement. Return an Encoder; the framework writes it.
type HandlerFunc func(ctx context.Context, r *http.Request) Encoder
