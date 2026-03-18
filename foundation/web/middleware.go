package web

// MidFunc wraps a HandlerFunc with additional behavior.
type MidFunc func(HandlerFunc) HandlerFunc

func wrapMiddleware(mw []MidFunc, handler HandlerFunc) HandlerFunc {
	for i := len(mw) - 1; i >= 0; i-- {
		handler = mw[i](handler)
	}
	return handler
}
