# Observation System (Frontend)

> Cross-cutting data collection system for recording task outcomes, debrief metadata, and contextual metrics. Observations are queryable by subject (task/context) or by kind (e.g., "debrief"). Data shape is flexible (untyped Record<string, unknown>) to accommodate various observation kinds without schema migrations. Currently used for task debrief outcomes and context-scoped analytics.

## Core Types

### Observation
```typescript
interface Observation {
  id: string
  subjectType: string          // 'task' | 'context'
  subjectId: string            // ID of the task or context
  kind: string                 // e.g., 'debrief', used to filter/group
  data: Record<string, unknown> // Flexible payload, shape varies by kind
  source: string               // e.g., 'user'
  confidence: number           // 0–1, reliability weighting
  weight: number               // Query weight factor
  createdAt: string            // ISO 8601 timestamp
}
```

### Debrief Observation Data (task outcomes)
```typescript
interface DebriefData {
  outcome: 'quick' | 'normal' | 'harder_than_expected' | 'blocked' | 'skipped'
  note?: string                // Optional user feedback
  contextId: string | null     // Task's parent context (for filtering)
  priority: string             // Task priority at debrief time
  energy: string               // Task energy level at debrief time
}
```

## File Map

### Services
- **`services/observationService.ts`** — **observationService** — API client for observations. Three methods: queryBySubject (fetch by task/context), queryByKind (fetch debrief/other kinds), record (POST observation).

## Usage

### Task Debrief Flow
`TaskDetailView.vue` → `TaskDebriefDialog.vue` → `observationService.record()`:
- User completes a task and is shown `TaskDebriefDialog`
- Dialog collects outcome + optional note
- Submission calls `observationService.record()` with debrief kind
- Recorded observation includes task priority, energy, context ID for later filtering

### Auto-Skip Logic
`TaskDetailView.vue` (line 97–100):
- On task load, calls `observationService.queryByKind('task', 'debrief')`
- Filters matching debrief observations by context/priority/energy
- Uses past outcomes to decide whether to auto-show next task or skip

### Context Observations Display
`ContextDetailView.vue` (line 45–46):
- Loads observations for a context with `observationService.queryBySubject('context', contextId)`
- Renders observations list in UI (what data appears depends on kind and context structure)

## Impact Callouts

### ⚠ Observation Interface
Changing the Observation shape (adding required fields, renaming fields, changing data types) affects:
- **`services/observationService.ts`** — API response deserialization and the `observationService` export
- **`views/TaskDetailView.vue`** — line 97–100, debrief filtering logic (currently only checks `o.data.contextId`)
- **`components/tasks/TaskDebriefDialog.vue`** — line 37, observation.record() payload shape
- **`views/ContextDetailView.vue`** — line 33 state type, line 45 fetch, line 395/549 template iteration

### ⚠ observationService.record() Payload Shape
Adding/removing fields in the record payload (debrief kind) affects:
- **`components/tasks/TaskDebriefDialog.vue`** — line 33–44 (success) and line 54–64 (skip) — payload construction
- **Backend validation/handling** — backend must accept/validate new fields in the kind-specific data
- **Future filtering logic** — if new fields are added (e.g., `duration`), any query that filters on them must be updated

### ⚠ queryByKind() Pagination/Size Limit
Currently hardcoded to `rows_per_page: 100` (line 31, observationService.ts):
- If debrief observations exceed 100, older ones are omitted from `shouldAutoSkip()` logic
- Changing the limit or adding pagination requires updates in `TaskDetailView.vue` (line 97)

## Cross-Domain Dependencies

- **task domain** — TaskDetailView, TaskDebriefDialog directly own the debrief flow; observations are strongly coupled to task/context IDs
- **context domain** — ContextDetailView displays observations scoped to a context
- **client service** — `services/client.ts` (HTTP request handler used by observationService)

## Backend Coupling

The frontend observation system is a thin proxy to the backend `/api/v1/observations` endpoints:
- `GET /api/v1/observations/{subjectType}/{subjectId}` — queryBySubject
- `GET /api/v1/observations?subject_type=...&kind=...` — queryByKind
- `POST /api/v1/observations` — record

Backend defines the Observation schema, validation, and data model. Frontend only serializes/deserializes.
