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
```

## File Map

### Stores
- `stores/transactionStore.ts` — **useTransactionStore** — Pinia store wrapping createCRUDStore (no create, since transactions come from import); adds `importing` flag, `lastImportResult`, `unreviewedCount` computed, `totalSpend` computed (sum of negative amounts), `importCSV(file, format?)`, `markReviewed(id)`

### Services
- `services/transactionService.ts` — **transactionService** — createCRUDService wrapper for `/api/v1/transactions`; extends with `importCSV(file, format?)` → POST `/api/v1/transactions/import` as multipart/form-data (uses raw fetch, not the client helper, because FormData cannot use JSON headers)

### Composables
No dedicated composable — TransactionBoardView uses the store directly.

### Components
- `components/transactions/TransactionFilterBar.vue` — **TransactionFilterBar** — filter UI for source, reviewed status, category, contextId
- `components/transactions/TransactionImport.vue` — **TransactionImport** — file picker + format selector; calls store.importCSV and shows ImportResult summary
- `components/transactions/TransactionRow.vue` — **TransactionRow** — single row in the transaction table; inline-editable cleanName/category/notes; emits update + mark-reviewed

### Views
- `views/TransactionBoardView.vue` — **TransactionBoardView** — renders TransactionFilterBar + table of TransactionRows + Pagination + TransactionImport modal; uses store directly

## Impact Callouts

### ⚠ Transaction (types/transaction.ts)
Changing this interface shape affects:
- `stores/transactionStore.ts` — `unreviewedCount` checks `.reviewed`; `totalSpend` reads `.amount`; `markReviewed` calls `update(id, { reviewed: true })`
- `services/transactionService.ts` — deserializes Transaction from list/getById responses
- `components/transactions/TransactionRow.vue` — binds .date, .description, .cleanName, .amount, .category, .contextId, .notes, .reviewed, .source
- `components/transactions/TransactionFilterBar.vue` — emits TransactionFilter with .contextId, .source, .reviewed, .category

### ⚠ UpdateTransaction (types/transaction.ts)
Changing updatable fields affects:
- `stores/transactionStore.ts` — `markReviewed` passes `{ reviewed }` partial; `update` passes full UpdateTransaction
- `services/transactionService.ts` — serializes UpdateTransaction as PUT body
- `components/transactions/TransactionRow.vue` — emits update events with UpdateTransaction-shaped payload

### ⚠ ImportResult (types/transaction.ts)
Changing this shape affects:
- `stores/transactionStore.ts` — stores in `lastImportResult` ref
- `services/transactionService.ts` — returns from importCSV; deserializes from multipart POST response
- `components/transactions/TransactionImport.vue` — displays .total, .imported, .skipped after upload

## Cross-Domain Dependencies

- `stores/contextStore.ts` — TransactionFilter.contextId links transactions to a context; TransactionRow may show context title (read from contextStore)
- `stores/toastStore.ts` — createCRUDStore (base) emits toasts for update/delete errors; importCSV errors are surfaced via thrown Error (component handles display)
- `services/client.ts` — all CRUD calls use the shared client; importCSV bypasses it to use raw fetch (FormData incompatibility with JSON Content-Type header)
