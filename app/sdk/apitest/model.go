// Package apitest provides support for executing API test logic.
package apitest

// TestAPIKey is the API key used for all test requests.
const TestAPIKey = "test-api-key-123"

// Table represents fields needed for running an HTTP API test.
type Table struct {
	Name       string
	URL        string
	Token      string
	Method     string
	StatusCode int
	Input      any
	GotResp    any
	ExpResp    any
	CmpFunc    func(got any, exp any) string
}
