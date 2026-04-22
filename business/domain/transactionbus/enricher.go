package transactionbus

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
)

// ExtractorEnricher adapts an extractor.Extractor to the Enricher interface.
type ExtractorEnricher struct {
	ext extractor.Extractor
}

// NewExtractorEnricher creates an enricher that wraps an Extractor.
func NewExtractorEnricher(ext extractor.Extractor) *ExtractorEnricher {
	return &ExtractorEnricher{ext: ext}
}

// EnrichTransaction calls ExtractText with typeHint="transaction" and maps
// the TextExtraction fields to TransactionEnrichment.
func (e *ExtractorEnricher) EnrichTransaction(ctx context.Context, txn Transaction) (TransactionEnrichment, error) {
	result, err := e.ext.ExtractText(ctx, txn.Description, "", nil, "transaction", 0, nil, nil)
	if err != nil {
		return TransactionEnrichment{}, fmt.Errorf("enrich transaction: %w", err)
	}

	enrichment := TransactionEnrichment{
		CleanName:         result.Summary,
		ContextConfidence: result.ContextConfidence,
	}

	if len(result.SuggestedContextKeywords) > 0 {
		enrichment.Category = result.SuggestedContextKeywords[0]
	}

	if result.SuggestedContextID != nil {
		id, err := uuid.Parse(*result.SuggestedContextID)
		if err == nil {
			enrichment.SuggestedContextID = &id
		}
	}

	return enrichment, nil
}
