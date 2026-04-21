# RawInput System (Frontend)

> RawInput manages the ingestion queue for unstructured text (emails, voice notes, clipboard pastes). Each input flows through a multi-step processing pipeline (sanitize → extract → match context → generate tasks/events/notes) and exposes status, errors, and pipeline step results. Users view the queue with filtering/pagination, inspect individual items with full step-by-step results, and trigger reprocessing for failed items. The system provides observability into enrichment pipeline failures.

## Core Types

```typescript
export interface StepResult {
  status: 'completed' | 'failed' | 'skipped'
  detail?: Record<string, unknown>
}

export interface PipelineResult {
  sanitize?: StepResult
  extraction?: StepResult
  contextMatch?: StepResult
  tasks?: StepResult
  events?: StepResult
  notes?: StepResult
}

export interface RawInput {
  id: string
  sourceType: string
  status: string
  rawContent: string
  processedAt?: string
  error?: string
  retryCount: number
  nextRetryAt?: string
  maxRetries: number
  result?: PipelineResult
  createdAt: string
}
```

## File Map

### Types
- `types/rawinput.ts` — **StepResult, PipelineResult, RawInput** — domain models (see rawinput-backend.md for full Go definitions)

### Stores
- `stores/rawinputStore.ts` — **useRawInputStore** — state: { items[], selectedItem, total, page, rowsPerPage, statusFilter, orderBy, orderDir, loading, error }, actions: fetchList(), fetchById(), reprocess(), setStatusFilter(), setPage(), setOrderBy()

### Services
- `services/rawinputService.ts` — **rawinputService** — list(params), getById(id), reprocess(id) wrapping /api/v1/raw-inputs endpoints

### Views
- `views/RawInputQueueView.vue` — route: `/ingest-queue`, displays paginated list of raw inputs with status badges (pending/processing/processed/partial/failed), filter dropdown, pagination controls; polls store.fetchList() every 15s; opens detail drawer on row click
- `views/RawInputDetailView.vue` — route: `/ingest-queue/:id`, shows single RawInput with raw content, pipelineSteps computed array (sanitize → extraction → contextMatch → tasks → events → notes), step result details, error messages, reprocess button

## Routes

| Path | Component | Parent / Child | Notes |
|------|-----------|---|---|
| `/ingest-queue` | RawInputQueueView | parent | Queue list with status filtering, pagination |
| `/ingest-queue/:id` | RawInputDetailView | child drawer | Detail view opened as drawer overlay |

## Impact Callouts

### ⚠ RawInput (types/rawinput.ts)
Changing this interface shape affects:
- `stores/rawinputStore.ts` — items[], selectedItem state; fetchList() and fetchById() unwrap responses; status filtering on items.value
- `services/rawinputService.ts` — list() and getById() response typing; QueryResult<RawInput> return type
- `views/RawInputQueueView.vue` — row binding to item properties (id, createdAt, status); statusBadgeClass(status) logic; isRetryScheduled(item) guards on nextRetryAt
- `views/RawInputDetailView.vue` — item.value property access (id, rawContent, error, status); pipelineSteps computed references item.value.result

### ⚠ PipelineResult (types/rawinput.ts)
Changing this interface affects:
- `views/RawInputDetailView.vue` — pipelineSteps computed array iteration; formatDetail(detail) on each step result object; step label/name constants hardcoded in array

### ⚠ StepResult (types/rawinput.ts)
Changing this interface affects:
- `views/RawInputDetailView.vue` — statusIcon(), statusColor() logic on status field; formatDetail() on detail field for rendering step outputs

## Cross-Domain Dependencies

### Outbound (this domain imports)
- `shared/DrawerPanel.vue` — RawInputQueueView uses for detail drawer layout
- `shared/LoadingSpinner.vue` — RawInputDetailView shows during fetch
- `shared/toastStore` — rawinputStore uses for error/success notifications

### Inbound (other domains import this)
- None currently (rawinput is leaf domain at UI level)
