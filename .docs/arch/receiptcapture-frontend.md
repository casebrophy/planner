# Receipt Capture Frontend Architecture

> Multi-step receipt capture workflow: camera → OCR text recognition → API extraction → manual review → optional split allocation. Stores receipt data and split assignments for transactions.

## Core Types

### ReceiptExtraction
Source: `web/src/services/receiptService.ts`

```ts
interface ReceiptExtraction {
  merchant: string
  date: string                    // ISO date string
  total: number                   // cents
  tax: number                     // cents
  subtotal: number                // cents
  items: Array<{
    description: string
    amount: number                // cents
    quantity: number
  }>
  notes?: string                  // optional extracted notes
}
```

### Split Types
Source: `web/src/types/split.ts`

```ts
interface Split {
  id: string
  transactionId: string
  partyName: string
  amount: number                  // cents
  venmoHandle?: string
  settled: boolean
  createdAt: string
  updatedAt: string
}

interface NewSplit {
  transactionId: string
  partyName: string
  amount: number                  // cents
  venmoHandle?: string
}

interface UpdateSplit {
  partyName?: string
  amount?: number
  venmoHandle?: string
  settled?: boolean
}
```

### Store State
Source: `web/src/stores/receiptCaptureStore.ts`

```ts
type CaptureStep = 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting' | 'confirmed'

interface ReceiptCaptureState {
  step: CaptureStep
  imageFile: File | null          // Raw image from camera
  ocrText: string                 // Tesseract.js OCR output
  extraction: ReceiptExtraction | null  // API-extracted receipt data
  splits: NewSplit[]              // User-entered split allocations
  error: string | null            // Step-level error message
}
```

### SplitEditor Internal
Source: `web/src/components/receipts/SplitEditor.vue`

```ts
interface SplitEntry {
  partyName: string
  amountDollars: string           // String for form binding
  venmoHandle: string
}
```

## File Map

### Stores
- `stores/receiptCaptureStore.ts` — **useReceiptCaptureStore** — Pinia store managing state machine (idle→scanning→extracting→reviewing→splitting→confirmed)

### Services
- `services/receiptService.ts` — **receiptService** — POST `/api/v1/transactions/extract-receipt` with OCR text, returns ReceiptExtraction
- `services/splitService.ts` — **splitService** — CRUD operations on Split records; extends createCRUDService; `getByTransaction(id)` queries splits for a transaction

### Composables
- `composables/useOCR.ts` — **useOCR** — Tesseract.js wrapper; `recognize(imageFile): Promise<string>` with progress tracking

### Components
- `components/receipts/ReceiptCamera.vue` — **ReceiptCamera** — File input capture; emits `@capture(file: File)`
- `components/receipts/ReceiptReview.vue` — **ReceiptReview** — Displays ReceiptExtraction; user edits merchant/date; emits `@update`, `@confirm`, `@addSplits`
- `components/receipts/SplitEditor.vue` — **SplitEditor** — Allocates totalCents among parties; computes remaining balance; generates Venmo deep links; emits `@done(splits: NewSplit[])`

### Views
- `views/ReceiptCaptureView.vue` — **ReceiptCaptureView** — Route `/receipts` orchestrating workflow; wires camera→OCR→extraction→review→split; delegates to router for navigation

## Impact Callouts

### ⚠ ReceiptExtraction
Changing the shape of extracted receipt data affects:
- `stores/receiptCaptureStore.ts` — stored in `extraction` state; passed to updateExtraction() and serialized on confirm
- `components/receipts/ReceiptReview.vue` — prop binding; template reads merchant, date, subtotal, tax, total, items array; emits partial updates
- `views/ReceiptCaptureView.vue` — destructured in `store.extraction?.total` to pass totalCents to SplitEditor; passed to receiptService
- `services/receiptService.ts` — return type from POST `/api/v1/transactions/extract-receipt`

### ⚠ NewSplit / UpdateSplit
Changing split data structure affects:
- `stores/receiptCaptureStore.ts` — stored in `splits` array; addSplit(), updateSplit(), removeSplit() mutate these
- `components/receipts/SplitEditor.vue` — emits `@done(splits: NewSplit[])` with array of these; internal mapping from SplitEntry → NewSplit in confirmSplits()
- `services/splitService.ts` — type parameter for CRUD operations; serialized in create() and update() requests
- `views/ReceiptCaptureView.vue` — receives emitted splits in handleConfirmWithSplits(); would pass to splitService.create()

### ⚠ CaptureStep (State Machine)
Changing workflow steps affects:
- `stores/receiptCaptureStore.ts` — step state and all transition methods (setImageFile→scanning, setOcrText→extracting, runExtraction→reviewing, startSplitting→splitting, confirm→confirmed)
- `views/ReceiptCaptureView.vue` — v-if conditionals on currentStep; step determines which component renders (ReceiptCamera, ReceiptReview, or SplitEditor)
- `components/receipts/ReceiptCaptureView.vue` — isProcessing display on 'extracting' step

### ⚠ ReceiptCaptureState
Changing overall store state shape affects:
- All components/views that call `useReceiptCaptureStore()` — binding to step, extraction, splits, error fields
- Navigation logic in handlers — confirm() and cancel() trigger router.push()

## State Transitions

```
idle
  ↓ (user captures image) → ReceiptCamera @capture
scanning
  ↓ (OCR processing) → useOCR.recognize()
extracting
  ↓ (API extracts data) → receiptService.extractReceipt()
reviewing
  ├─ (user edits merchant/date) → ReceiptReview @update
  ├─ (user clicks "Looks good") → ReceiptReview @confirm → router.push('/transactions')
  └─ (user clicks "Split with others") → ReceiptReview @addSplits → startSplitting()
splitting
  ├─ (user confirms splits) → SplitEditor @done → router.push('/transactions')
  └─ (user cancels) → reset() → router.push('/transactions')
confirmed
  ↓ (navigate away)
idle
```

## Cross-Domain Dependencies

### Incoming (features that depend on this)
- **Entry point:** `views/CaptureView.vue` (navigation hub) — "Receipt" button routes to `/receipts`
- **Transaction creation flow** (not yet implemented) — after splits confirmed, would call `splitService.create()` and navigate to `/transactions`

### Outgoing (external dependencies)
- `services/client.ts` — request() helper for API calls
- `services/createCRUDService.ts` — factory for splitService
- Vue Router — navigation via `router.push()`
- Tesseract.js (npm) — OCR worker via useOCR
- Pinia — state management
- Tailwind CSS — dark theme styling

## Styling & Theme

Consistent dark theme across all components:
- Page: `bg-gray-900` (near-black)
- Cards/sections: `bg-gray-800` (dark gray)
- Borders: `border-gray-700` (medium gray)
- Text: `text-white` (light text), `text-gray-400` (labels)
- Actions: `bg-blue-600` (primary), `bg-green-600` (confirm), `bg-red-900` (danger/remove)

## Testing Notes

- `ReceiptCamera` — mock file input; verify @capture emits File
- `ReceiptReview` — mock ReceiptExtraction prop; verify @update emits partial changes; verify buttons emit correct events
- `SplitEditor` — verify split evenly divides by (numParties + 1); verify remaining balance calculation; verify Venmo URL encoding
- `receiptCaptureStore` — verify state transitions; verify error handling on extraction failure
- `useOCR` — mock Tesseract.js worker; verify recognize() returns text; verify progress updates
