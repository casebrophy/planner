# Receipt Capture — Implementation Plan

**Goal:** Snap a photo of a receipt, OCR it in the browser, extract structured data via AI, review/edit, create a transaction with expense splits, and generate Venmo deep links for splitting.

**Architecture:** Browser-side OCR (Tesseract.js) extracts raw text from the camera image — no image leaves the device. The raw text is sent to the backend extractor pipeline (new `ExtractReceipt` method) which returns structured receipt data. User reviews/edits, confirms, and a transaction + splits are created. Splits are stored in a `transaction_splits` table.

**Key decisions:**
- No image storage anywhere — OCR runs entirely in the browser
- No Capacitor — HTML5 `<input type="file" accept="image/*" capture="environment">` for camera access
- Splits live in their own table (`transaction_splits`) with FK to `transactions`
- Venmo integration is frontend-only (deep links, no API)
- Receipt extraction reuses the existing `Extractor` interface (new method) + `TieredRouter`

---

## File Map

### Backend — new files

| File | Responsibility |
|------|---------------|
| `business/domain/splitbus/model.go` | `Split`, `NewSplit`, `UpdateSplit` structs |
| `business/domain/splitbus/splitbus.go` | Business methods + `Storer` interface |
| `business/domain/splitbus/filter.go` | `QueryFilter` struct (by transaction_id) |
| `business/domain/splitbus/order.go` | `OrderBy` constants |
| `business/domain/splitbus/stores/splitdb/model.go` | DB struct + converters |
| `business/domain/splitbus/stores/splitdb/splitdb.go` | SQL queries |
| `business/domain/splitbus/stores/splitdb/filter.go` | `applyFilter()` |
| `business/domain/splitbus/stores/splitdb/order.go` | `orderByClause()` |
| `app/domain/splitapp/model.go` | App DTOs + converters |
| `app/domain/splitapp/splitapp.go` | Handlers: create, queryByTransaction, update, delete |
| `app/domain/splitapp/route.go` | Route registration |
| `app/domain/splitapp/filter.go` | `parseFilter()` |
| `app/domain/splitapp/order.go` | `parseOrder()` |

### Backend — modified files

| File | Change |
|------|--------|
| `business/sdk/migrate/sql/migrate.sql` | Add `transaction_splits` table (v1.35) |
| `business/domain/ingestbus/extractor/model.go` | Add `ReceiptExtraction` struct + `ExtractReceipt` to `Extractor` interface |
| `business/domain/ingestbus/extractor/claudecli.go` | Implement `ExtractReceipt` on `ClaudeCLIExtractor` |
| `business/domain/ingestbus/extractor/ollama.go` | Implement `ExtractReceipt` on `OllamaExtractor` |
| `business/domain/ingestbus/extractor/failover.go` | Implement `ExtractReceipt` on `FailoverExtractor` |
| `business/domain/ingestbus/extractor/router.go` | Route `ExtractReceipt` in `TieredRouter` (receipts → general, not localOnly) |
| `business/domain/ingestbus/extractor/mock.go` | Add `ExtractReceipt` to mock |
| `app/domain/transactionapp/transactionapp.go` | Add `extractReceipt` handler (receives OCR text, returns structured receipt data) |
| `app/domain/transactionapp/model.go` | Add `ReceiptExtractionRequest`, `AppReceiptExtraction` DTOs |
| `app/domain/transactionapp/route.go` | Register `POST /api/v1/transactions/extract-receipt` |
| `api/services/planner/main.go` | Wire `splitapp.Routes{}` into mux |

### Frontend — new files

| File | Responsibility |
|------|---------------|
| `web/src/types/split.ts` | `Split`, `NewSplit`, `UpdateSplit` TypeScript types |
| `web/src/services/splitService.ts` | API client for splits (CRUD by transaction) |
| `web/src/services/receiptService.ts` | `extractReceipt(ocrText)` — POST OCR text, get structured data back |
| `web/src/stores/receiptCaptureStore.ts` | Pinia store: OCR state, extraction result, split editor state |
| `web/src/composables/useOCR.ts` | Tesseract.js wrapper — `recognize(imageFile)` → raw text |
| `web/src/components/receipts/ReceiptCamera.vue` | HTML5 camera input + image preview |
| `web/src/components/receipts/ReceiptReview.vue` | Editable extracted fields (merchant, date, total, items) |
| `web/src/components/receipts/SplitEditor.vue` | Add/remove split parties, amounts, Venmo deep links |
| `web/src/views/ReceiptCaptureView.vue` | Route-level view: capture → OCR → extract → review → confirm |

### Frontend — modified files

| File | Change |
|------|--------|
| `web/src/types/index.ts` | Export split types |
| `web/src/router/index.ts` | Add `/receipts` route |
| `web/src/views/CaptureView.vue` | Add "Receipt" mode tab alongside task/context |
| `web/src/stores/captureStore.ts` | Add `'receipt'` to `CaptureMode` type |
| `web/src/composables/useCapture.ts` | Handle receipt mode → navigate to ReceiptCaptureView |

---

## Task 1: Migration — transaction_splits table

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Add transaction_splits table DDL**

```sql
-- Version: 1.35
-- Description: Create transaction_splits table
CREATE TABLE transaction_splits (
    split_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    transaction_id UUID        NOT NULL,
    party_name     TEXT        NOT NULL,
    amount         INTEGER     NOT NULL,  -- cents
    venmo_handle   TEXT,
    settled        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (split_id),
    FOREIGN KEY (transaction_id) REFERENCES transactions(transaction_id) ON DELETE CASCADE
);

CREATE INDEX idx_transaction_splits_transaction_id ON transaction_splits(transaction_id);
```

---

## Task 2: Split domain — business + store layers

**Files:**
- Create: `business/domain/splitbus/model.go`
- Create: `business/domain/splitbus/splitbus.go`
- Create: `business/domain/splitbus/filter.go`
- Create: `business/domain/splitbus/order.go`
- Create: `business/domain/splitbus/stores/splitdb/model.go`
- Create: `business/domain/splitbus/stores/splitdb/splitdb.go`
- Create: `business/domain/splitbus/stores/splitdb/filter.go`
- Create: `business/domain/splitbus/stores/splitdb/order.go`

**Pattern to follow:** `transactionbus` + `transactiondb` — same 3-layer shape.

- [ ] **Step 1: Business model** (`splitbus/model.go`)

```go
type Split struct {
    ID            uuid.UUID
    TransactionID uuid.UUID
    PartyName     string
    Amount        int       // cents
    VenmoHandle   *string
    Settled       bool
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

type NewSplit struct {
    TransactionID uuid.UUID
    PartyName     string
    Amount        int
    VenmoHandle   *string
}

type UpdateSplit struct {
    PartyName   *string
    Amount      *int
    VenmoHandle *string
    Settled     *bool
}
```

- [ ] **Step 2: Storer interface + Business methods** (`splitbus/splitbus.go`)

Methods: `Create`, `Update`, `Delete`, `QueryByID`, `Query` (filtered by transaction_id), `Count`, `DeleteByTransaction`

- [ ] **Step 3: QueryFilter and OrderBy** (`splitbus/filter.go`, `splitbus/order.go`)

Filter by `TransactionID` (required). Order by `created_at` (default).

- [ ] **Step 4: Store implementation** (`splitdb/`)

Standard SQL CRUD. `applyFilter` always includes `transaction_id` WHERE clause. DB struct with `db:"column"` tags, `toDBSplit`/`toBusSplit` converters.

- [ ] **Step 5: Tests**

Store integration tests using `dbtest` — create transaction first, then CRUD splits.

---

## Task 3: Split domain — app layer + wiring

**Files:**
- Create: `app/domain/splitapp/model.go`
- Create: `app/domain/splitapp/splitapp.go`
- Create: `app/domain/splitapp/route.go`
- Create: `app/domain/splitapp/filter.go`
- Create: `app/domain/splitapp/order.go`
- Modify: `api/services/planner/main.go`

- [ ] **Step 1: App DTOs** (`splitapp/model.go`)

```go
type Split struct {
    ID            string  `json:"id"`
    TransactionID string  `json:"transactionId"`
    PartyName     string  `json:"partyName"`
    Amount        int     `json:"amount"`
    VenmoHandle   *string `json:"venmoHandle,omitempty"`
    Settled       bool    `json:"settled"`
    CreatedAt     string  `json:"createdAt"`
    UpdatedAt     string  `json:"updatedAt"`
}

type NewSplit struct {
    TransactionID string  `json:"transactionId"`
    PartyName     string  `json:"partyName"`
    Amount        int     `json:"amount"`
    VenmoHandle   *string `json:"venmoHandle,omitempty"`
}
```

- [ ] **Step 2: Handlers** (`splitapp/splitapp.go`)

Handlers: `create`, `queryByTransaction`, `update`, `delete`

- [ ] **Step 3: Routes** (`splitapp/route.go`)

```go
func (Routes) Add(a *web.App, cfg mux.Config) {
    splitStore := splitdb.NewStore(cfg.Log, cfg.DB)
    splitBus := splitbus.NewBusiness(cfg.Log, splitStore)
    hdl := &app{splitBus: splitBus}
    authen := mid.Auth(cfg.APIKey)

    a.Handle(http.MethodPost,   "/api/v1/splits", hdl.create, authen)
    a.Handle(http.MethodGet,    "/api/v1/transactions/{transaction_id}/splits", hdl.queryByTransaction, authen)
    a.Handle(http.MethodPut,    "/api/v1/splits/{split_id}", hdl.update, authen)
    a.Handle(http.MethodDelete, "/api/v1/splits/{split_id}", hdl.delete, authen)
}
```

- [ ] **Step 4: Wire into main.go**

Add `splitapp.Routes{}` to the mux route adders.

- [ ] **Step 5: API integration tests**

Test create split on transaction, query splits, update settled status, delete.

---

## Task 4: Receipt extraction — backend extractor

**Files:**
- Modify: `business/domain/ingestbus/extractor/model.go`
- Modify: `business/domain/ingestbus/extractor/claudecli.go`
- Modify: `business/domain/ingestbus/extractor/ollama.go`
- Modify: `business/domain/ingestbus/extractor/failover.go`
- Modify: `business/domain/ingestbus/extractor/router.go`
- Modify: `business/domain/ingestbus/extractor/mock.go`

- [ ] **Step 1: Add ReceiptExtraction model** (`model.go`)

```go
// ExtractReceipt parses OCR text from a receipt image into structured data.
ExtractReceipt(ctx context.Context, ocrText string) (ReceiptExtraction, error)

type ReceiptExtraction struct {
    Merchant  string              `json:"merchant"`
    Date      string              `json:"date"`       // YYYY-MM-DD
    Total     int                 `json:"total"`      // cents
    Tax       int                 `json:"tax"`        // cents
    Subtotal  int                 `json:"subtotal"`   // cents
    Items     []ReceiptLineItem   `json:"items"`
    Notes     string              `json:"notes,omitempty"`
}

type ReceiptLineItem struct {
    Description string `json:"description"`
    Amount      int    `json:"amount"` // cents
    Quantity    int    `json:"quantity"`
}
```

- [ ] **Step 2: Implement on ClaudeCLIExtractor** (`claudecli.go`)

New prompt + JSON schema for receipt extraction. Claude parses the OCR text into the structured format. Key prompt considerations:
- Handle OCR noise (misspelled words, broken lines)
- Amounts in cents (multiply by 100)
- Date normalization to YYYY-MM-DD

- [ ] **Step 3: Implement on OllamaExtractor** (`ollama.go`)

Same prompt pattern adapted for Ollama.

- [ ] **Step 4: Implement on FailoverExtractor** (`failover.go`)

Standard failover: try primary, fall back to secondary.

- [ ] **Step 5: Route in TieredRouter** (`router.go`)

Route `ExtractReceipt` to `general` extractor (not localOnly — receipt text isn't sensitive financial data, it's merchant names and totals).

- [ ] **Step 6: Update mock** (`mock.go`)

Add `ExtractReceipt` to mock with configurable return.

- [ ] **Step 7: Tests**

Unit test the prompt builder with sample OCR text. Test that TieredRouter routes to general.

---

## Task 5: Receipt extraction — app endpoint

**Files:**
- Modify: `app/domain/transactionapp/transactionapp.go`
- Modify: `app/domain/transactionapp/model.go`
- Modify: `app/domain/transactionapp/route.go`

- [ ] **Step 1: Add request/response DTOs** (`model.go`)

```go
type ReceiptExtractionRequest struct {
    OCRText string `json:"ocrText"`
}

type AppReceiptExtraction struct {
    Merchant string                `json:"merchant"`
    Date     string                `json:"date"`
    Total    int                   `json:"total"`
    Tax      int                   `json:"tax"`
    Subtotal int                   `json:"subtotal"`
    Items    []AppReceiptLineItem  `json:"items"`
    Notes    string                `json:"notes,omitempty"`
}

type AppReceiptLineItem struct {
    Description string `json:"description"`
    Amount      int    `json:"amount"`
    Quantity    int    `json:"quantity"`
}
```

- [ ] **Step 2: Add extractReceipt handler** (`transactionapp.go`)

```go
func (a *app) extractReceipt(ctx context.Context, r *http.Request) web.Encoder {
    var req ReceiptExtractionRequest
    if err := web.Decode(r, &req); err != nil {
        return errs.New(errs.InvalidArgument, err)
    }
    extraction, err := a.extractor.ExtractReceipt(ctx, req.OCRText)
    if err != nil {
        return errs.Newf(errs.Internal, "extract receipt: %s", err)
    }
    return toAppReceiptExtraction(extraction)
}
```

The `app` struct needs access to the extractor. Add it via the route config (extractor already on `mux.Config`).

- [ ] **Step 3: Register route** (`route.go`)

```go
a.Handle(http.MethodPost, "/api/v1/transactions/extract-receipt", hdl.extractReceipt, authen)
```

- [ ] **Step 4: API integration test**

Test with sample OCR text → verify structured extraction response.

---

## Task 6: Frontend — OCR composable + receipt service

**Files:**
- Create: `web/src/composables/useOCR.ts`
- Create: `web/src/services/receiptService.ts`
- Modify: `web/package.json` (add `tesseract.js` dependency)

- [ ] **Step 1: Install Tesseract.js**

```bash
cd api/services/frontend/web && npm install tesseract.js
```

- [ ] **Step 2: useOCR composable** (`composables/useOCR.ts`)

```ts
import { ref } from 'vue'
import { createWorker } from 'tesseract.js'

export function useOCR() {
  const isProcessing = ref(false)
  const progress = ref(0)
  const error = ref<string | null>(null)

  async function recognize(imageFile: File): Promise<string> {
    isProcessing.value = true
    progress.value = 0
    error.value = null
    try {
      const worker = await createWorker('eng', 1, {
        logger: (m) => { if (m.status === 'recognizing text') progress.value = m.progress }
      })
      const { data: { text } } = await worker.recognize(imageFile)
      await worker.terminate()
      return text
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'OCR failed'
      throw e
    } finally {
      isProcessing.value = false
    }
  }

  return { recognize, isProcessing, progress, error }
}
```

- [ ] **Step 3: receiptService** (`services/receiptService.ts`)

```ts
import { request } from './client'

export interface ReceiptExtraction {
  merchant: string
  date: string
  total: number
  tax: number
  subtotal: number
  items: { description: string; amount: number; quantity: number }[]
  notes?: string
}

export const receiptService = {
  extractReceipt(ocrText: string): Promise<ReceiptExtraction> {
    return request<ReceiptExtraction>('/api/v1/transactions/extract-receipt', {
      method: 'POST',
      body: JSON.stringify({ ocrText }),
    })
  },
}
```

---

## Task 7: Frontend — split types + service + store

**Files:**
- Create: `web/src/types/split.ts`
- Create: `web/src/services/splitService.ts`
- Modify: `web/src/types/index.ts`

- [ ] **Step 1: Split types** (`types/split.ts`)

```ts
export interface Split {
  id: string
  transactionId: string
  partyName: string
  amount: number
  venmoHandle?: string
  settled: boolean
  createdAt: string
  updatedAt: string
}

export interface NewSplit {
  transactionId: string
  partyName: string
  amount: number
  venmoHandle?: string
}

export interface UpdateSplit {
  partyName?: string
  amount?: number
  venmoHandle?: string
  settled?: boolean
}
```

- [ ] **Step 2: Split service** (`services/splitService.ts`)

CRUD service for splits. `getByTransaction(transactionId)` uses `GET /api/v1/transactions/:id/splits`.

- [ ] **Step 3: Export from index**

Add split type exports to `types/index.ts`.

---

## Task 8: Frontend — receipt capture view + components

**Files:**
- Create: `web/src/stores/receiptCaptureStore.ts`
- Create: `web/src/components/receipts/ReceiptCamera.vue`
- Create: `web/src/components/receipts/ReceiptReview.vue`
- Create: `web/src/components/receipts/SplitEditor.vue`
- Create: `web/src/views/ReceiptCaptureView.vue`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/views/CaptureView.vue`
- Modify: `web/src/stores/captureStore.ts`
- Modify: `web/src/composables/useCapture.ts`

- [ ] **Step 1: receiptCaptureStore** (`stores/receiptCaptureStore.ts`)

State machine: `idle` → `scanning` → `extracting` → `reviewing` → `splitting` → `confirmed`

```ts
interface ReceiptCaptureState {
  step: 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting' | 'confirmed'
  imageFile: File | null
  ocrText: string
  extraction: ReceiptExtraction | null
  splits: NewSplit[]
  error: string | null
}
```

Actions: `startScan`, `processOCR`, `extractReceipt`, `updateExtraction`, `addSplit`, `removeSplit`, `confirm` (creates transaction + splits).

- [ ] **Step 2: ReceiptCamera component**

HTML5 file input with `accept="image/*"` and `capture="environment"`. Shows image preview after selection. Triggers OCR on selection.

- [ ] **Step 3: ReceiptReview component**

Displays and allows editing of extracted receipt fields: merchant, date, total, tax, line items. "Looks good" button advances to split step (or skips to confirm if no splits needed).

- [ ] **Step 4: SplitEditor component**

- Add party: name, amount, optional Venmo handle
- Remaining amount auto-calculated (total - sum of splits = "your share")
- Venmo deep link button per party: `venmo://paycharge?txn=charge&recipients=<handle>&amount=<dollars>&note=<merchant>`
- Quick split buttons: "Split evenly" (divides total by N parties)

- [ ] **Step 5: ReceiptCaptureView**

Multi-step view orchestrating the flow:
1. Camera capture (ReceiptCamera)
2. OCR processing (progress bar from useOCR)
3. AI extraction (loading state while backend processes)
4. Review extracted data (ReceiptReview)
5. Optional: add splits (SplitEditor)
6. Confirm → creates transaction via transactionStore.create + splits via splitService

- [ ] **Step 6: Router + CaptureView integration**

Add `/receipts` route pointing to `ReceiptCaptureView`. Add "Receipt" tab to CaptureView mode toggle — selecting it navigates to `/receipts`.

- [ ] **Step 7: Frontend tests**

Vitest tests for:
- `useOCR` composable (mock Tesseract worker)
- `receiptCaptureStore` state transitions
- `SplitEditor` math (amounts sum to total, even split calculation)
- Venmo deep link generation

---

## Ordering Constraints

```
Task 1 (migration) → Task 2 (split business) → Task 3 (split app)
Task 1 (migration) → Task 4 (extractor) → Task 5 (extract endpoint)
Task 6 (OCR + receipt service) depends on Task 5
Task 7 (split types/service) depends on Task 3
Task 8 (UI) depends on Tasks 5, 6, 7
```

Parallelizable pairs:
- Tasks 2+4 (split domain + extractor — independent backend work)
- Tasks 6+7 (OCR composable + split types — independent frontend work)
