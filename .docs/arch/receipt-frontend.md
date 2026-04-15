# Receipt Capture System

> Mobile-first receipt capture flow: user photographs a receipt, Tesseract.js runs client-side OCR, the backend extracts structured data (merchant, date, line items, totals), and the user reviews/edits before optionally splitting the expense with others via Venmo deep links. State machine driven by `CaptureStep`: idle → scanning → extracting → reviewing → splitting → confirmed.

## Core Types

### ReceiptExtraction (`services/receiptService.ts`)
```ts
export interface ReceiptExtraction {
  merchant: string
  date: string
  total: number       // cents
  tax: number         // cents
  subtotal: number    // cents
  items: { description: string; amount: number; quantity: number }[]
  notes?: string
}
```

### CaptureStep + ReceiptCaptureState (`stores/receiptCaptureStore.ts`)
```ts
export type CaptureStep = 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting' | 'confirmed'

export interface ReceiptCaptureState {
  step: CaptureStep
  imageFile: File | null
  ocrText: string
  extraction: ReceiptExtraction | null
  splits: NewSplit[]
  error: string | null
}
```

### NewSplit (`types/split.ts` — cross-domain, shared with transaction/split domain)
```ts
export interface NewSplit {
  transactionId: string
  partyName: string
  amount: number        // cents
  venmoHandle?: string
}
```

## File Map

### Stores
- `stores/receiptCaptureStore.ts` — **useReceiptCaptureStore** — Pinia store managing the capture state machine (step transitions, extraction result, splits list). Actions: `reset`, `setImageFile`, `setOcrText`, `runExtraction`, `updateExtraction`, `startSplitting`, `addSplit`, `removeSplit`, `updateSplit`, `confirm`.

### Services
- `services/receiptService.ts` — **receiptService** — API client for `POST /api/v1/transactions/extract-receipt`. Sends OCR text, returns `ReceiptExtraction`.

### Composables
- `composables/useOCR.ts` — **useOCR** — Client-side OCR via Tesseract.js. Returns `{ recognize, isProcessing, progress, error }`. Used only by `ReceiptCaptureView`.

### Components
- `components/receipts/ReceiptCamera.vue` — **ReceiptCamera** — File input with camera capture and image preview. Emits `capture(file: File)`.
- `components/receipts/ReceiptReview.vue` — **ReceiptReview** — Editable review of extracted receipt (merchant, date, totals, line items). Props: `extraction: ReceiptExtraction`. Emits: `update(Partial<ReceiptExtraction>)`, `confirm`, `addSplits`.
- `components/receipts/SplitEditor.vue` — **SplitEditor** — Multi-party expense splitter with even-split and Venmo deep links. Props: `totalCents: number`, `transactionId: string`. Emits: `done(splits: NewSplit[])`.

### Views
- `views/CaptureView.vue` — **CaptureView** — Hub view with buttons to navigate to task, context, or receipt capture.
- `views/ReceiptCaptureView.vue` — **ReceiptCaptureView** — Orchestrates the full capture flow: wires `ReceiptCamera` → `useOCR` → `receiptService` → `ReceiptReview` → `SplitEditor`. Consumes `useReceiptCaptureStore` and `useOCR`.

### Routes
- `/capture` → `CaptureView`
- `/receipts` → `ReceiptCaptureView`

## Impact Callouts

### ⚠ ReceiptExtraction (`services/receiptService.ts`)
Changing this interface shape affects:
- `stores/receiptCaptureStore.ts` — `extraction` ref typed as `ReceiptExtraction | null`; `updateExtraction()` spreads `Partial<ReceiptExtraction>`
- `views/ReceiptCaptureView.vue` — `handleUpdateExtraction()` accepts `Partial<ReceiptExtraction>`; passes `store.extraction` as prop to `ReceiptReview`
- `components/receipts/ReceiptReview.vue` — prop type `extraction: ReceiptExtraction`; template binds `.merchant`, `.date`, `.subtotal`, `.tax`, `.total`, `.items[]` (`.description`, `.amount`); emits `Partial<ReceiptExtraction>` on field edits

### ⚠ CaptureStep (`stores/receiptCaptureStore.ts`)
Changing this union affects:
- `views/ReceiptCaptureView.vue` — `currentStep` computed used in `v-if` guards for each flow phase (`idle`, `scanning`, `extracting`, `reviewing`, `splitting`)
- `stores/receiptCaptureStore.ts` — all action methods set `step.value` to specific literals

### ⚠ NewSplit (`types/split.ts`)
Changing this interface shape affects:
- `stores/receiptCaptureStore.ts` — `splits` ref typed as `NewSplit[]`; `addSplit()`, `updateSplit()` manipulate the array
- `components/receipts/SplitEditor.vue` — `confirmSplits()` maps local `SplitEntry[]` → `NewSplit[]` (reads `transactionId`, `partyName`, `amount`, `venmoHandle`); emits `done(NewSplit[])`
- `views/ReceiptCaptureView.vue` — `handleConfirmWithSplits()` receives `NewSplit[]`

## Cross-Domain Dependencies

- `types/split.ts` — `NewSplit` interface shared with the transaction/split domain
- `composables/useOCR.ts` — general-purpose OCR composable (not receipt-specific), depends on `tesseract.js`
- `services/client.ts` — shared HTTP `request()` function used by `receiptService`
- `router/index.ts` — lazy-loads `CaptureView` at `/capture` and `ReceiptCaptureView` at `/receipts`
