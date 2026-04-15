# RawInput System

> RawInput manages the ingestion pipeline for unstructured text (emails, voice notes, clipboard pastes). Each raw input flows through a multi-step processing pipeline (sanitize → extract → match context → generate tasks/events/notes) and tracks status and errors. Users view the queue, inspect failures, and trigger reprocessing. The system provides observability into the enrichment pipeline.

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

### Stores
- `stores/rawinputStore.ts` — **useRawInputStore** — custom store (not CRUD) managing list, pagination, filtering by status, ordering, selection, reprocess action

### Services
- `services/rawinputService.ts` — **rawinputService** — CRUD + custom actions (list, reprocess) for /api/v1/raw-inputs endpoint

### Components
- `components/shared/ProcessingStatus.vue` — stepper UI showing pending → processing → processed pipeline progress; handles error (red X) and partial (orange warning) states with distinct visual indicators

### Views
- `views/RawInputQueueView.vue` — displays paginated queue of raw inputs with status filtering (pending, processing, processed, partial, failed) and reprocess buttons for failed/partial/pending items
- `views/RawInputDetailView.vue` — shows single raw input with full pipeline result breakdown, retry history, error details

## Impact Callouts

### ⚠ RawInput (types/rawinput.ts)
Changing this interface shape affects:
- `stores/rawinputStore.ts` — fetchList/fetchById operations, status filtering, orderBy state management
- `services/rawinputService.ts` — list() query params (status, sourceType, orderBy), reprocess() POST endpoint
- `views/RawInputQueueView.vue` — displays status badge, createdAt, rowsPerPage pagination
- `views/RawInputDetailView.vue` — renders full RawInput with rawContent, PipelineResult breakdown per step, error messages

### ⚠ PipelineResult (types/rawinput.ts)
Changing this interface affects:
- `views/RawInputDetailView.vue` — step-by-step status display, detail object rendering (extraction results, matched tasks, etc.)

## Cross-Domain Dependencies

- **task, event, note domains** — RawInput generates these via extraction pipeline
- **clarification domain** — unresolved extractions may create clarification tasks
- **ollama service** — extraction step runs via ollama models for ML-powered entity/intent recognition
