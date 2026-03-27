// Package unitest provides support for executing unit test logic.
package unitest

import "context"

// Table represent fields needed for running a unit test.
type Table struct {
	Name    string
	ExpResp any
	ExcFunc func(ctx context.Context) any
	CmpFunc func(got any, exp any) string
}
