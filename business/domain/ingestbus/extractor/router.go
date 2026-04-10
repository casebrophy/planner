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
func (r *TieredRouter) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error) {
	return r.general.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
}

// ExtractText routes based on typeHint:
//   - typeHint == "transaction" → localOnly (Ollama)
//   - everything else → general
//
// When localOnly is nil and typeHint is "transaction", returns a zero
// TextExtraction (enrichment skipped) without error.
func (r *TieredRouter) ExtractText(ctx context.Context, text string, activeContexts []ContextRef, typeHint string) (TextExtraction, error) {
	if typeHint == "transaction" {
		if r.localOnly == nil {
			r.log.Info(ctx, "tiered_router", "status", "enrichment skipped, ollama not configured")
			return TextExtraction{}, nil
		}
		return r.localOnly.ExtractText(ctx, text, activeContexts, typeHint)
	}

	return r.general.ExtractText(ctx, text, activeContexts, typeHint)
}
