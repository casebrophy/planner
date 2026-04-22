package extractor

import (
	"context"
	"strings"

	"github.com/casebrophy/planner/foundation/logger"
)

// FailoverExtractor tries primary, falls back to fallback only on qualifying errors.
type FailoverExtractor struct {
	log      *logger.Logger
	primary  Extractor
	fallback Extractor
}

// NewFailoverExtractor accepts concrete types to prevent accidental nesting.
// Internally stores as Extractor interface for testability.
func NewFailoverExtractor(log *logger.Logger, primary *ClaudeCodeExtractor, fallback *OllamaExtractor) *FailoverExtractor {
	return &FailoverExtractor{
		log:      log,
		primary:  primary,
		fallback: fallback,
	}
}

// newFailoverExtractorForTest is a package-private helper for tests that bypasses
// the concrete-type constraint of the public constructor.
func newFailoverExtractorForTest(log *logger.Logger, primary Extractor, fallback Extractor) *FailoverExtractor {
	return &FailoverExtractor{
		log:      log,
		primary:  primary,
		fallback: fallback,
	}
}

// isFallbackError reports whether err should trigger the Ollama fallback.
// Triggers on:
//   - error message contains "429"
//   - error message contains "context" AND "limit"
//   - error message contains "connection" OR "timeout" OR "refused"
//   - error message contains "502" OR "401" OR "login" OR "exit status 1"
//
// Does NOT trigger for 400, schema errors, etc.
func isFallbackError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())

	if strings.Contains(msg, "429") {
		return true
	}
	if strings.Contains(msg, "context") && strings.Contains(msg, "limit") {
		return true
	}
	if strings.Contains(msg, "connection") || strings.Contains(msg, "timeout") || strings.Contains(msg, "refused") {
		return true
	}

	// Sidecar / CLI availability errors — auth expired, gateway failure, CLI crash.
	if strings.Contains(msg, "502") || strings.Contains(msg, "401") || strings.Contains(msg, "login") || strings.Contains(msg, "exit status 1") {
		return true
	}

	return false
}

// ExtractEmail tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error) {
	result, err := f.primary.ExtractEmail(ctx, subject, bodyText, fromAddress, userCorrection, activeContexts)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractEmail(ctx, subject, bodyText, fromAddress, userCorrection, activeContexts)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}

// ExtractText tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string, typeHintConfidence float64, candidates []EntityMatch, contextAnnotations []string) (TextExtraction, error) {
	result, err := f.primary.ExtractText(ctx, text, userCorrection, activeContexts, typeHint, typeHintConfidence, candidates, contextAnnotations)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractText(ctx, text, userCorrection, activeContexts, typeHint, typeHintConfidence, candidates, contextAnnotations)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}

// ExtractReceipt tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error) {
	result, err := f.primary.ExtractReceipt(ctx, ocrText)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return ReceiptExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractReceipt(ctx, ocrText)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return ReceiptExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}

// AnalyzeGaps tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error) {
	result, err := f.primary.AnalyzeGaps(ctx, entityType, entityContent, relatedEntities)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return GapAnalysis{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama for gap analysis", "error", err.Error())

	result, err = f.fallback.AnalyzeGaps(ctx, entityType, entityContent, relatedEntities)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed for gap analysis", "error", err.Error())
		return GapAnalysis{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded for gap analysis")
	return result, nil
}
