package extractor_test

import (
	"context"
	"os"
	"testing"

	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/foundation/logger"
)

// trackingMock wraps MockExtractor and records whether it was called.
type trackingMock struct {
	extractor.MockExtractor
	emailCalled bool
	textCalled  bool
}

func (t *trackingMock) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []extractor.ContextRef) (extractor.EmailExtraction, error) {
	t.emailCalled = true
	return t.MockExtractor.ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts)
}

func (t *trackingMock) ExtractText(ctx context.Context, text string, activeContexts []extractor.ContextRef, typeHint string) (extractor.TextExtraction, error) {
	t.textCalled = true
	return t.MockExtractor.ExtractText(ctx, text, activeContexts, typeHint)
}

func testLogger() *logger.Logger {
	return logger.New(os.Stdout, logger.LevelInfo, "test")
}

func TestTieredRouter_ExtractText_Transaction(t *testing.T) {
	log := testLogger()
	general := &trackingMock{MockExtractor: extractor.MockExtractor{TextResult: extractor.TextExtraction{Summary: "general"}}}
	local := &trackingMock{MockExtractor: extractor.MockExtractor{TextResult: extractor.TextExtraction{Summary: "local"}}}

	router := extractor.NewTieredRouter(log, general, local)
	result, err := router.ExtractText(context.Background(), "test", nil, "transaction")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "local" {
		t.Errorf("expected local result, got %q", result.Summary)
	}
	if !local.textCalled {
		t.Error("localOnly extractor was not called")
	}
	if general.textCalled {
		t.Error("general extractor should not be called for transactions")
	}
}

func TestTieredRouter_ExtractText_NonTransaction(t *testing.T) {
	log := testLogger()
	general := &trackingMock{MockExtractor: extractor.MockExtractor{TextResult: extractor.TextExtraction{Summary: "general"}}}
	local := &trackingMock{MockExtractor: extractor.MockExtractor{TextResult: extractor.TextExtraction{Summary: "local"}}}

	router := extractor.NewTieredRouter(log, general, local)
	result, err := router.ExtractText(context.Background(), "test", nil, "voice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "general" {
		t.Errorf("expected general result, got %q", result.Summary)
	}
	if !general.textCalled {
		t.Error("general extractor was not called")
	}
	if local.textCalled {
		t.Error("localOnly extractor should not be called for non-transactions")
	}
}

func TestTieredRouter_ExtractEmail_AlwaysGeneral(t *testing.T) {
	log := testLogger()
	general := &trackingMock{MockExtractor: extractor.MockExtractor{Result: extractor.EmailExtraction{Summary: "general"}}}
	local := &trackingMock{MockExtractor: extractor.MockExtractor{Result: extractor.EmailExtraction{Summary: "local"}}}

	router := extractor.NewTieredRouter(log, general, local)
	result, err := router.ExtractEmail(context.Background(), "subject", "body", "from@test.com", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "general" {
		t.Errorf("expected general result, got %q", result.Summary)
	}
	if !general.emailCalled {
		t.Error("general extractor was not called")
	}
	if local.emailCalled {
		t.Error("localOnly extractor should not be called for email")
	}
}

func TestTieredRouter_ExtractText_NilLocalOnly(t *testing.T) {
	log := testLogger()
	general := &trackingMock{MockExtractor: extractor.MockExtractor{TextResult: extractor.TextExtraction{Summary: "general"}}}

	router := extractor.NewTieredRouter(log, general, nil)
	result, err := router.ExtractText(context.Background(), "test", nil, "transaction")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Summary != "" {
		t.Errorf("expected empty result for skipped enrichment, got %q", result.Summary)
	}
	if general.textCalled {
		t.Error("general extractor should not be called for transaction when local is nil")
	}
}
