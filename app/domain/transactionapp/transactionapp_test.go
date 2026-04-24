package transactionapp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/casebrophy/planner/app/domain/transactionapp"
	"github.com/casebrophy/planner/app/sdk/apitest"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/foundation/logger"
	"github.com/casebrophy/planner/foundation/web"
)

// MockExtractor implements the extractor.Extractor interface for testing.
type MockExtractor struct {
	receiptData extractor.ReceiptExtraction
	err         error
}

func (m *MockExtractor) ExtractEmail(ctx context.Context, subject, bodyText, fromAddress, userCorrection string, activeContexts []extractor.ContextRef) (extractor.EmailExtraction, error) {
	return extractor.EmailExtraction{}, nil
}

func (m *MockExtractor) ExtractText(ctx context.Context, text, userCorrection string, activeContexts []extractor.ContextRef, typeHint string, typeHintConfidence float64, candidates []extractor.EntityMatch, contextAnnotations []string) (extractor.TextExtraction, error) {
	return extractor.TextExtraction{}, nil
}

func (m *MockExtractor) ExtractReceipt(ctx context.Context, ocrText string) (extractor.ReceiptExtraction, error) {
	return m.receiptData, m.err
}

func (m *MockExtractor) AnalyzeGaps(ctx context.Context, entityType, entityContent string, relatedEntities []extractor.RelatedEntity) (extractor.GapAnalysis, error) {
	return extractor.GapAnalysis{}, nil
}

func TestExtractReceipt(t *testing.T) {
	tests := []struct {
		name           string
		ocrText        string
		mockReceipt    extractor.ReceiptExtraction
		mockErr        error
		expectedStatus int
		expectedResp   *transactionapp.AppReceiptExtraction
	}{
		{
			name:    "successful extraction",
			ocrText: "Whole Foods\n2026-04-14\n$25.50",
			mockReceipt: extractor.ReceiptExtraction{
				Merchant: "Whole Foods",
				Date:     "2026-04-14",
				Total:    2550,
				Tax:      200,
				Subtotal: 2350,
				Items: []extractor.ReceiptLineItem{
					{
						Description: "Milk",
						Amount:      450,
						Quantity:    1,
					},
					{
						Description: "Bread",
						Amount:      300,
						Quantity:    2,
					},
				},
				Notes: "Thank you for your purchase",
			},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedResp: &transactionapp.AppReceiptExtraction{
				Merchant: "Whole Foods",
				Date:     "2026-04-14",
				Total:    2550,
				Tax:      200,
				Subtotal: 2350,
				Items: []transactionapp.AppReceiptLineItem{
					{
						Description: "Milk",
						Amount:      450,
						Quantity:    1,
					},
					{
						Description: "Bread",
						Amount:      300,
						Quantity:    2,
					},
				},
				Notes: "Thank you for your purchase",
			},
		},
		{
			name:    "no items receipt",
			ocrText: "Target\n2026-04-13\n$15.99",
			mockReceipt: extractor.ReceiptExtraction{
				Merchant: "Target",
				Date:     "2026-04-13",
				Total:    1599,
				Tax:      120,
				Subtotal: 1479,
				Items:    []extractor.ReceiptLineItem{},
			},
			mockErr:        nil,
			expectedStatus: http.StatusOK,
			expectedResp: &transactionapp.AppReceiptExtraction{
				Merchant: "Target",
				Date:     "2026-04-13",
				Total:    1599,
				Tax:      120,
				Subtotal: 1479,
				Items:    []transactionapp.AppReceiptLineItem{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create logger
			var buf bytes.Buffer
			log := logger.New(&buf, logger.LevelInfo, "TEST")

			// Create a minimal web app with just the transaction routes
			cfg := mux.Config{
				Log:       log,
				APIKey:    apitest.TestAPIKey,
				Extractor: &MockExtractor{receiptData: tt.mockReceipt, err: tt.mockErr},
			}

			app := web.NewApp(log)
			transactionapp.Routes{}.Add(app, cfg)

			// Create request body
			type requestBody struct {
				OCRText string `json:"ocrText"`
			}
			body, _ := json.Marshal(requestBody{OCRText: tt.ocrText})

			// Make the request
			req, err := http.NewRequest(http.MethodPost, "/api/v1/transactions/extract-receipt", bytes.NewBuffer(body))
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-API-Key", apitest.TestAPIKey)

			// Record the response
			w := httptest.NewRecorder()
			app.ServeHTTP(w, req)

			// Check status code
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectedStatus == http.StatusOK {
				var got transactionapp.AppReceiptExtraction
				if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
					t.Fatalf("Failed to unmarshal response: %v", err)
				}

				if diff := cmp.Diff(&got, tt.expectedResp); diff != "" {
					t.Errorf("Response mismatch: %s", diff)
				}
			}
		})
	}
}
