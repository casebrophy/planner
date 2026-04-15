# Transaction System (Frontend)

> Financial transaction domain: import bank CSV exports, categorize transactions, assign to contexts, and mark as reviewed. Transactions are read-only after import except for cleanName/category/contextId/notes/reviewed fields. The board view is a filterable table; CSV import is via a modal dialog.

## Core Types

```ts
// types/transaction.ts
interface Transaction {
  id: string
  rawInputId?: string
  source: string          // e.g. 'chase', 'amex'
  date: string
  description: string     // raw bank description
  cleanName?: string      // user-edited clean name
  amount: number          // negative = spend, positive = credit
  category?: string
  contextId?: string
  notes?: string
  reviewed: boolean
  createdAt: string
}

interface UpdateTransaction {
  cleanName?: string
  category?: string
  contextId?: string
  notes?: string
  reviewed?: boolean
}

interface TransactionFilter {
  contextId?: string
  source?: string
  reviewed?: boolean
  category?: string
}

interface ImportResult {
  total: number
  imported: number
  skipped: number
}

interface EnrichmentStatus {
  active: number
  pending: number
  done: number
  failed: number
  enabled: boolean
}
```

## File Map

### Stores
- `stores/transactionStore.ts` — **useTransactionStore** — Pinia store wrapping createCRUDStore (no create, since transactions come from import); adds `importing` flag, `lastImportResult`, `enrichmentStatus` ref, `unreviewedCount` computed, `totalSpend` computed (sum of negative amounts), `importCSV(file, format?)`, `markReviewed(id)`, `startEnrichmentPolling()`/`stopEnrichmentPolling()` (3s interval)

### Services
- `services/transactionService.ts` — **transactionService** — createCRUDService wrapper for `/api/v1/transactions`; extends with `importCSV(file, format?)` → POST `/api/v1/transactions/import` as multipart/form-data (uses raw fetch, not the client helper, because FormData cannot use JSON headers); `getEnrichmentStatus()` → GET `/api/v1/transactions/enrichment-status`

### Composables
No dedicated composable — TransactionBoardView uses the store directly.

### Components
- `components/transactions/EnrichmentStatusBar.vue` — **EnrichmentStatusBar** — live status indicator showing active/pending/done/failed enrichment counts; pulsing green dot when processing, gray when idle
- `components/transactions/TransactionFilterBar.vue` — **TransactionFilterBar** — filter UI for reviewed status (All/Needs Review/Reviewed) and source (chase_checking/chase_credit/amex); updates store filter and refetches list
- `components/transactions/TransactionImport.vue` — **TransactionImport** — file picker + format selector; calls store.importCSV and shows ImportResult summary
- `components/transactions/TransactionRow.vue` — **TransactionRow** — single row in the transaction table; displays date, cleanName/description, category badge, amount (with color), reviewed status; emits 'review' (id) and 'click' (id) events

### Views
- `views/TransactionBoardView.vue` — **TransactionBoardView** — renders TransactionFilterBar + EnrichmentStatusBar + table of TransactionRows + Pagination + TransactionImport modal; starts/stops enrichment polling on mount/unmount

## Impact Callouts

### ⚠ Transaction (types/transaction.ts)
Changing this interface shape affects:
- `stores/transactionStore.ts` — `unreviewedCount` checks `.reviewed`; `totalSpend` reads `.amount`; `markReviewed` calls `update(id, { reviewed: true })`
- `services/transactionService.ts` — deserializes Transaction from list/getById responses
- `components/transactions/TransactionRow.vue` — binds .date, .description, .cleanName, .amount, .category, .reviewed in template; passes to review/click emitters
- `components/transactions/TransactionFilterBar.vue` — emits TransactionFilter with .source, .reviewed fields only
- `views/TransactionBoardView.vue` — uses .total, .totalSpend (computed from .amount), .unreviewedCount (computed from .reviewed)

### ⚠ UpdateTransaction (types/transaction.ts)
Changing updatable fields affects:
- `stores/transactionStore.ts` — `markReviewed` passes `{ reviewed }` partial; `update` passes full UpdateTransaction
- `services/transactionService.ts` — serializes UpdateTransaction as PUT body

### ⚠ ImportResult (types/transaction.ts)
Changing this shape affects:
- `stores/transactionStore.ts` — stores in `lastImportResult` ref
- `services/transactionService.ts` — returns from importCSV; deserializes from multipart POST response
- `components/transactions/TransactionImport.vue` — displays .total, .imported, .skipped after upload

## Cross-Domain Dependencies

- `stores/toastStore.ts` — TransactionImport uses toastStore to display import success/error messages; createCRUDStore (base) emits toasts for update/delete errors
- `services/client.ts` — all CRUD calls use the shared client; importCSV bypasses it to use raw fetch (FormData incompatibility with JSON Content-Type header)
- `components/layout/PageHeader.vue`, `components/shared/LoadingSpinner.vue`, `components/shared/EmptyState.vue`, `components/shared/Pagination.vue` — layout and UI shared components used by TransactionBoardView
