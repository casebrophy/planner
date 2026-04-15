package extractor_test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/foundation/logger"
)

func TestBuildReceiptExtractionPrompt(t *testing.T) {
	ocrText := "WHOLE FOODS\n$25.50\nTax: $2.00"
	prompt := extractor.BuildReceiptExtractionPrompt(ocrText)

	if !strings.Contains(prompt, ocrText) {
		t.Errorf("expected prompt to contain OCR text, got: %s", prompt)
	}

	if !strings.Contains(prompt, "cents") {
		t.Errorf("expected prompt to contain cents rules")
	}

	if !strings.Contains(prompt, "YYYY-MM-DD") {
		t.Errorf("expected prompt to contain date format rule")
	}

	if !strings.Contains(prompt, "merchant") {
		t.Errorf("expected prompt to contain merchant field")
	}

	if !strings.Contains(prompt, "JSON") {
		t.Errorf("expected prompt to contain JSON instruction")
	}
}

func TestTieredRouterExtractReceipt(t *testing.T) {
	log := logger.New(os.Stdout, logger.LevelInfo, "test")

	// Mock extractors with different results
	general := &extractor.MockExtractor{
		ReceiptResult: extractor.ReceiptExtraction{
			Merchant: "General Store",
			Total:    5000,
		},
	}
	localOnly := &extractor.MockExtractor{
		ReceiptResult: extractor.ReceiptExtraction{
			Merchant: "Local Store",
			Total:    3000,
		},
	}

	router := extractor.NewTieredRouter(log, general, localOnly)
	result, err := router.ExtractReceipt(context.Background(), "sample receipt text")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should route to general, not localOnly
	if result.Merchant != "General Store" {
		t.Errorf("expected general result with merchant 'General Store', got %q", result.Merchant)
	}

	if result.Total != 5000 {
		t.Errorf("expected total 5000, got %d", result.Total)
	}
}
