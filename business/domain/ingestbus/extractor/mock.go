package extractor

import "context"

// MockExtractor implements Extractor for testing.
type MockExtractor struct {
	Result         EmailExtraction
	TextResult     TextExtraction
	ReceiptResult  ReceiptExtraction
	GapResult      GapAnalysis
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
func (m *MockExtractor) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string, candidates []EntityMatch, contextAnnotations []string) (TextExtraction, error) {
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

// AnalyzeGaps returns the configured gap result or error.
func (m *MockExtractor) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error) {
	if m.Err != nil {
		return GapAnalysis{}, m.Err
	}
	return m.GapResult, nil
}
