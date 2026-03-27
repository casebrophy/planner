package extractor

import "context"

// MockExtractor implements Extractor for testing.
type MockExtractor struct {
	Result EmailExtraction
	Err    error
}

// ExtractEmail returns the configured result or error.
func (m *MockExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	if m.Err != nil {
		return EmailExtraction{}, m.Err
	}
	return m.Result, nil
}
