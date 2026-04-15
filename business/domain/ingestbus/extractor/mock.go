package extractor

import "context"

// MockExtractor implements Extractor for testing.
type MockExtractor struct {
	Result         EmailExtraction
	TextResult     TextExtraction
	ReceiptResult  ReceiptExtraction
	Err            error
}

// ExtractEmail returns the configured result or error.
func (m *MockExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error) {
	if m.Err != nil {
		return EmailExtraction{}, m.Err
	}
	return m.Result, nil
}

// ExtractText returns the configured text result or error.
func (m *MockExtractor) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	if m.Err != nil {
		return TextExtraction{}, m.Err
	}
	return m.TextResult, nil
}

// ExtractReceipt returns the configured receipt result or error.
func (m *MockExtractor) ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error) {
	if m.Err != nil {
		return ReceiptExtraction{}, m.Err
	}
	return m.ReceiptResult, nil
}
