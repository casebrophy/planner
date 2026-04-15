# Capture System

> Multi-mode capture hub for receipts and quick entries. Focuses on receipt capture with OCR extraction, manual review/editing, and expense splitting. Includes navigation to other capture modes (Task, Context).

## Core Types

### ReceiptExtraction (receiptService.ts)
```typescript
interface ReceiptExtraction {
  merchant: string
  date: string
  total: number        // cents
  tax: number          // cents
  subtotal: number     // cents
  items: { 
    description: string
    amount: number     // cents
    quantity: number 
  }[]
  notes?: string
}
```

### CaptureStep (receiptCaptureStore.ts)
```typescript
type CaptureStep = 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting' | 'confirmed'
```

### ReceiptCaptureState (receiptCaptureStore.ts)
```typescript
interface ReceiptCaptureState {
  step: CaptureStep
  imageFile: File | null
  ocrText: string
  extraction: ReceiptExtraction | null
  splits: NewSplit[]
  error: string | null
}
```

### Split Types (split.ts)
```typescript
interface NewSplit {
  transactionId: string
  partyName: string
  amount: number        // cents
  venmoHandle?: string
}
```

## File Map

### Stores
- `stores/receiptCaptureStore.ts` — **useReceiptCaptureStore()** — Pinia store managing receipt capture state machine (step, imageFile, ocrText, extraction, splits, error). Exposes actions for workflow: setImageFile, setOcrText, runExtraction, updateExtraction, startSplitting, addSplit, removeSplit, updateSplit, confirm, reset.

### Services
- `services/receiptService.ts` — **receiptService.extractReceipt()** — POST `/api/v1/transactions/extract-receipt` with ocrText, returns ReceiptExtraction. Uses client-side request wrapper.

### Composables
- `composables/useOCR.ts` — **useOCR()** — Tesseract.js-based OCR worker. Returns { recognize(imageFile), isProcessing, progress, error }. Runs text recognition on image File and returns extracted text string.

### Components
- `components/receipts/ReceiptCamera.vue` — **ReceiptCamera** — File input with preview. Props: none. Emits: `capture(file: File)`.
- `components/receipts/ReceiptReview.vue` — **ReceiptReview** — Edit extracted receipt fields (merchant, date). Displays totals summary and line items. Props: `extraction: ReceiptExtraction`. Emits: `update(data: Partial<ReceiptExtraction>)`, `confirm()`, `addSplits()`, `cancel()`.
- `components/receipts/SplitEditor.vue` — **SplitEditor** — Multi-party expense split UI. Add parties, input names/amounts/Venmo handles, split evenly, generate Venmo deep links. Props: `totalCents: number`, `transactionId: string`. Emits: `done(splits: NewSplit[])`.

### Views
- `views/CaptureView.vue` — **CaptureView** — Navigation hub at `/capture`. Three buttons (Task, Context, Receipt) route to `/tasks`, `/contexts`, `/receipts`. No state or store dependencies.
- `views/ReceiptCaptureView.vue` — **ReceiptCaptureView** — Main receipt capture workflow at `/receipts`. Orchestrates multi-step flow (camera → OCR → extraction → review → splitting → confirm). Integrates store, useOCR, and all receipt components.

## Impact Callouts

### ⚠ ReceiptExtraction (receiptService.ts)
Changing this interface affects:
- `useReceiptCaptureStore` — extraction ref stores entire object; updateExtraction merges Partial updates
- `ReceiptCaptureView` — passes extraction to ReceiptReview and accesses `extraction.total` for SplitEditor
- `ReceiptReview` — receives extraction as prop, reads merchant/date/subtotal/tax/total/items fields, emits updates via 'update' event
- `SplitEditor` — consumes `extraction.total` for dollar/cent calculations and split validation

### ⚠ CaptureStep (receiptCaptureStore.ts)
Changing state machine affects:
- `useReceiptCaptureStore` — step is core state; all action methods transition step value
- `ReceiptCaptureView` — `currentStep` computed prop controls which component renders (Camera if idle/scanning, progress if extracting, ReceiptReview if reviewing, SplitEditor if splitting)

### ⚠ NewSplit (split.ts)
Changing this type affects:
- `SplitEditor` — builds NewSplit objects from local SplitEntry state in confirmSplits(); reads transactionId, partyName, amount (converted cents), venmoHandle
- `useReceiptCaptureStore` — splits ref is array<NewSplit>; addSplit, removeSplit, updateSplit mutate this array

### ⚠ ReceiptCamera (ReceiptCamera.vue)
Changing component interface affects:
- `ReceiptCaptureView` — imports and renders ReceiptCamera when step is idle or scanning; listens to @capture emit, calls handleCapture(file)

### ⚠ ReceiptReview (ReceiptReview.vue)
Changing component interface affects:
- `ReceiptCaptureView` — renders when step is reviewing; passes extraction prop, listens to @update/@confirm/@addSplits/@cancel emits

### ⚠ SplitEditor (SplitEditor.vue)
Changing component interface affects:
- `ReceiptCaptureView` — renders when step is splitting; passes totalCents (from extraction.total) and transactionId props; listens to @done emit, calls handleConfirmWithSplits(splits)

## Cross-Domain Dependencies

**Inbound:**
- `services/client.ts` — receiptService uses request() HTTP client wrapper
- `types/split.ts` — NewSplit type imported by SplitEditor and receiptCaptureStore
- `router/index.ts` — CaptureView and ReceiptCaptureView registered as routes `/capture` and `/receipts`

**Outbound:**
- ReceiptCaptureView routes to `/transactions` on confirm (router.push)
- Could integrate with transactionService (future: persist created transactions from splits)
- Could integrate with contextService/taskService (future: link split transactions to contexts/tasks)

## Workflow Summary

1. **CaptureView** (`/capture`) — User selects Receipt capture mode
2. **ReceiptCaptureView** (`/receipts`) enters flow:
   - **Idle/Scanning** — ReceiptCamera captures image, emits File
   - **Extracting** — useOCR.recognize() extracts text; receiptService.extractReceipt() POST ocrText to backend
   - **Reviewing** — ReceiptReview displays extracted fields, allows edit (merchant, date), user confirms or chooses to split
   - **Splitting** — SplitEditor builds multi-party split, calculates Venmo links
   - **Confirmed** — Navigate to `/transactions`

State persists in useReceiptCaptureStore until reset() called (on cancel or confirm).
