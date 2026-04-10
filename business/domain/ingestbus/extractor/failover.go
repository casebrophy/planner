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
//
// Does NOT trigger for 400, 401, 500, schema errors, etc.
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
	return false
}

// ExtractEmail tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	result, err := f.primary.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return EmailExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}

// ExtractText tries primary; if isFallbackError, logs and tries fallback; else returns error as-is.
func (f *FailoverExtractor) ExtractText(ctx context.Context, text string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	result, err := f.primary.ExtractText(ctx, text, activeContexts, typeHint)
	if err == nil {
		return result, nil
	}

	if !isFallbackError(err) {
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "claude failed, falling back to ollama", "error", err.Error())

	result, err = f.fallback.ExtractText(ctx, text, activeContexts, typeHint)
	if err != nil {
		f.log.Error(ctx, "extractor", "status", "ollama fallback failed", "error", err.Error())
		return TextExtraction{}, err
	}

	f.log.Info(ctx, "extractor", "status", "ollama fallback succeeded")
	return result, nil
}
