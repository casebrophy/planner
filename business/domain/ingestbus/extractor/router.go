package extractor

import (
	"context"

	"github.com/casebrophy/planner/foundation/logger"
)

// TieredRouter routes extraction calls based on sensitivity tier.
// Financial data (transactions) → localOnly extractor (Ollama).
// Everything else → general extractor (FailoverExtractor or Claude).
type TieredRouter struct {
	log       *logger.Logger
	general   Extractor
	localOnly Extractor // nil when Ollama is disabled
}

// NewTieredRouter creates a TieredRouter. localOnly may be nil; in that case,
// transaction requests return a zero TextExtraction (enrichment skipped).
func NewTieredRouter(log *logger.Logger, general Extractor, localOnly Extractor) *TieredRouter {
	return &TieredRouter{
		log:       log,
		general:   general,
		localOnly: localOnly,
	}
}

// ExtractEmail always routes to the general extractor (emails are not sensitive).
func (r *TieredRouter) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []ContextRef) (EmailExtraction, error) {
	return r.general.ExtractEmail(ctx, subject, bodyText, fromAddress, userCorrection, activeContexts)
}

// ExtractText routes based on typeHint:
//   - typeHint == "transaction" → localOnly (Ollama)
//   - everything else → general
//
// When localOnly is nil and typeHint is "transaction", returns a zero
// TextExtraction (enrichment skipped) without error.
func (r *TieredRouter) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	if typeHint == "transaction" {
		if r.localOnly == nil {
			r.log.Info(ctx, "tiered_router", "status", "enrichment skipped, ollama not configured")
			return TextExtraction{}, nil
		}
		return r.localOnly.ExtractText(ctx, text, userCorrection, activeContexts, typeHint)
	}

	return r.general.ExtractText(ctx, text, userCorrection, activeContexts, typeHint)
}

// ExtractReceipt routes to the general extractor.
// Receipt text (merchant names, totals) is not sensitive financial data.
func (r *TieredRouter) ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error) {
	return r.general.ExtractReceipt(ctx, ocrText)
}

// AnalyzeGaps routes to general extractor (Claude) first, falls back to localOnly (Ollama) if unavailable.
// Gap analysis now routes to Claude first to avoid 180s Ollama timeout; financial data filtering
// happens at call sites via Transaction source check in ingestbus.
func (r *TieredRouter) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []RelatedEntity) (GapAnalysis, error) {
	if r.general != nil {
		return r.general.AnalyzeGaps(ctx, entityType, entityContent, relatedEntities)
	}
	if r.localOnly != nil {
		return r.localOnly.AnalyzeGaps(ctx, entityType, entityContent, relatedEntities)
	}
	return GapAnalysis{}, nil
}
