# ReceiptCapture Frontend Architecture

## Overview

Multi-step receipt capture UI orchestrating camera → OCR → extraction → split management.

## Components

### ReceiptCaptureView.vue
Route-level view managing capture workflow state transitions.

**Props:** None  
**Emits:** None (navigation via router)

**State:**
- Uses `useReceiptCaptureStore()` for step management
- Uses `useOCR()` for OCR processing

**Flow:**
1. `idle/scanning` → `<ReceiptCamera>` (user takes photo)
2. `extracting` → Shows progress spinner during receipt extraction
3. `reviewing` → `<ReceiptReview>` (user confirms extracted data)
4. `splitting` → `<SplitEditor>` (user allocates expense)

**Routes:**
- `/receipts` (ReceiptCaptureView)

### SplitEditor.vue
Component for managing expense splits among parties.

**Props:**
```ts
totalCents: number        // Total transaction amount in cents
transactionId: string     // ID to associate splits with transaction
```

**Emits:**
```ts
done: [splits: NewSplit[]]  // User confirms splits
```

**Features:**
- Display remaining unallocated amount
- Per-party entry (name, amount, optional Venmo handle)
- "Add party" button
- "Split evenly" button (divides by numParties + 1 including "you")
- Venmo deep links per party: `venmo://paycharge?txn=charge&recipients={handle}&amount={dollars}&note=`
- "Confirm splits" button (validates all entries have name + amount)

**Internal State:**
```ts
splits: Array<{
  partyName: string
  amountDollars: string
  venmoHandle: string
}>
```

### CaptureView.vue
Navigation hub for capture modes.

**Routes:**
- `/capture` (CaptureView)

**Features:**
- Three buttons: Task, Context, Receipt
- Receipt button navigates to `/receipts`

## Stores (Created by Parallel Worker)

### receiptCaptureStore
Pinia store managing receipt capture workflow.

**State:**
```ts
step: 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting'
imageFile: File | null
ocrText: string
extraction: ReceiptExtraction | null  // From receiptService
splits: NewSplit[]
error: string | null
```

**Methods:**
- `setImageFile(file)` → transitions to 'scanning'
- `setOcrText(text)`
- `runExtraction()` → calls receiptService.extractReceipt(), transitions to 'reviewing'
- `updateExtraction(data)` → merges into extraction
- `startSplitting()` → transitions to 'splitting'
- `confirm()` → transitions to 'idle'
- `reset()` → clears all state

## Service Integration

### receiptService
- `extractReceipt(imageFile, ocrText): Promise<ReceiptExtraction>`
  Returns merchant, date, total (cents), tax, subtotal, items array

### splitService
- `create(splits: NewSplit[]): Promise<Split[]>`
  Creates split records (used after transaction created)

## Type Dependencies

```ts
ReceiptExtraction {
  merchant: string
  date: string
  total: number        // cents
  tax: number          // cents
  subtotal: number     // cents
  items: Array<{
    description: string
    amount: number     // cents
    quantity: number
  }>
}

NewSplit {
  transactionId: string
  partyName: string
  amount: number       // cents
  venmoHandle?: string
}
```

## Styling

Dark theme using Tailwind CSS:
- `bg-gray-900` (page background)
- `bg-gray-800` (cards)
- `border-gray-700` (borders)
- `text-white` (text)
- `bg-blue-600` (primary actions)
- `bg-green-600` (confirm)
- `bg-red-900` (danger)

## Testing

File: `__tests__/receiptCapture.test.ts`

Tests:
- Store state transitions
- Split evenly math
- Venmo URL format

Uses mocked `receiptService.extractReceipt()`.
