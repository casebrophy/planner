# Raw Input Pipeline Tracking Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-step pipeline result tracking to raw inputs so users can see exactly what happened during ingestion (which steps ran, what was extracted, what was created).

**Architecture:** Add a `result JSONB` column to `raw_inputs`. The ingest pipeline builds a `PipelineResult` struct as it executes, writing it before marking processed/failed. Frontend gets a detail drawer on the existing RawInputQueueView.

**Tech Stack:** Go (backend), PostgreSQL (JSONB column), Vue 3 + TypeScript + Pinia (frontend), Vitest (frontend tests)

**Spec:** `.docs/superpowers/specs/2026-04-07-rawinput-pipeline-tracking-design.md`

---

### Task 1: Migration — add `result` column

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Add migration 1.22**

Append to the end of `business/sdk/migrate/sql/migrate.sql`:

```sql
-- Version: 1.22
-- Description: Add pipeline result tracking to raw_inputs
ALTER TABLE raw_inputs ADD COLUMN result JSONB;
```

- [ ] **Step 2: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add result JSONB column to raw_inputs (migration 1.22)"
```

---

### Task 2: Backend — add `Result` field across all layers

**Files:**
- Modify: `business/domain/rawinputbus/model.go`
- Modify: `business/domain/rawinputbus/stores/rawinputdb/model.go`
- Modify: `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`
- Modify: `app/domain/rawinputapp/model.go`

- [ ] **Step 1: Add Result to business model**

In `business/domain/rawinputbus/model.go`, add `encoding/json` to imports and add the field to both structs:

Add to `RawInput` struct after `MaxRetries int`:
```go
Result     json.RawMessage
```

Add to `UpdateRawInput` struct after `NextRetryAt *time.Time`:
```go
Result     *json.RawMessage
```

- [ ] **Step 2: Add Result to DB model and converters**

In `business/domain/rawinputbus/stores/rawinputdb/model.go`, add `encoding/json` to imports.

Add to `rawInputDB` struct after `MaxRetries int`:
```go
Result     json.RawMessage `db:"result"`
```

In `toDBRawInput`, add after `MaxRetries`:
```go
Result:     ri.Result,
```

In `toBusRawInput`, add after `MaxRetries`:
```go
Result:     ri.Result,
```

- [ ] **Step 3: Add Result to SQL queries**

In `business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go`:

**Create** — add `result` to the INSERT column list and VALUES:
```sql
(raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at)
VALUES
(:raw_input_id, :source_type, :status, :raw_content, :processed_at, :error, :retry_count, :next_retry_at, :max_retries, :result, :created_at)
```

**Update** — add `result = :result` to the SET clause:
```sql
UPDATE raw_inputs SET
    status = :status,
    processed_at = :processed_at,
    error = :error,
    retry_count = :retry_count,
    next_retry_at = :next_retry_at,
    result = :result
WHERE
    raw_input_id = :raw_input_id
```

**All SELECT queries** (Query, QueryByID, QueryRetryable, ResetForReprocess) — add `result` to the column list. Every `SELECT` that lists columns explicitly needs `result` added. Search for each SELECT statement and add `result` after `max_retries`. There are 4 SELECT statements total:
1. `Query` method (line ~69): `SELECT raw_input_id, source_type, status, raw_content, processed_at, error, retry_count, next_retry_at, max_retries, result, created_at`
2. `QueryByID` method (line ~113): same column list
3. `QueryRetryable` method (line ~129): same column list
4. `ResetForReprocess` method (line ~152): same column list in RETURNING clause

- [ ] **Step 4: Add Result to rawinputbus Update method**

In `business/domain/rawinputbus/rawinputbus.go`, in the `Update` method, add after the `NextRetryAt` block:
```go
if uri.Result != nil {
    ri.Result = *uri.Result
}
```

- [ ] **Step 5: Add Result to app DTO**

In `app/domain/rawinputapp/model.go`, add `encoding/json` to imports (already imported).

Add to `RawInput` struct after `MaxRetries`:
```go
Result      json.RawMessage `json:"result,omitempty"`
```

In `toAppRawInput`, add after `MaxRetries`:
```go
Result:     ri.Result,
```

- [ ] **Step 6: Verify it compiles**

```bash
cd /Users/casebrophy/personal/planner && go build ./...
```

- [ ] **Step 7: Run existing tests**

```bash
make test
```

- [ ] **Step 8: Commit**

```bash
git add business/domain/rawinputbus/model.go business/domain/rawinputbus/rawinputbus.go business/domain/rawinputbus/stores/rawinputdb/model.go business/domain/rawinputbus/stores/rawinputdb/rawinputdb.go app/domain/rawinputapp/model.go
git commit -m "feat: add result JSONB field to rawinputbus across all layers"
```

---

### Task 3: Backend — define PipelineResult types and instrument processTextInput

**Files:**
- Modify: `business/domain/ingestbus/ingestbus.go`

- [ ] **Step 1: Add PipelineResult and StepResult types**

Add these types after the `IngestResult` struct (around line 38) in `business/domain/ingestbus/ingestbus.go`:

```go
// PipelineResult tracks per-step pipeline outcomes for observability.
type PipelineResult struct {
	Sanitize     *StepResult `json:"sanitize,omitempty"`
	Extraction   *StepResult `json:"extraction,omitempty"`
	ContextMatch *StepResult `json:"contextMatch,omitempty"`
	Tasks        *StepResult `json:"tasks,omitempty"`
	Events       *StepResult `json:"events,omitempty"`
	Notes        *StepResult `json:"notes,omitempty"`
}

// StepResult records the outcome of a single pipeline step.
type StepResult struct {
	Status string         `json:"status"` // "completed", "failed", "skipped"
	Detail map[string]any `json:"detail,omitempty"`
}
```

- [ ] **Step 2: Fix double MarkProcessing in processTextInput**

In `processTextInput` (starts ~line 497), remove lines 498-502 (the duplicate MarkProcessing call):

```go
// Step 2: Mark as processing
ri, err := b.rawInputBus.MarkProcessing(ctx, ri)
if err != nil {
    return IngestResult{}, fmt.Errorf("mark processing: %w", err)
}
```

Delete these 5 lines. The `ri` parameter already has the correct state because `ProcessRawInputByID` calls `MarkProcessing` before delegating.

Also update the function signature to accept `ri` as a value (not reassigned):
The `ri` parameter is already passed by value, so just remove the `:=` reassignment block.

- [ ] **Step 3: Instrument processTextInput with PipelineResult**

Replace the body of `processTextInput` (after removing the MarkProcessing block) with pipeline result tracking. The key changes are:

**After Step 4 (sanitize), around line 530:**
Add after the PII log block:
```go
pr := PipelineResult{}
pr.Sanitize = &StepResult{
    Status: "completed",
    Detail: map[string]any{"pii_findings": len(sanitizeResult.Findings)},
}
```

**Step 5 (extraction) — on error path (around line 534):**
Replace the existing soft-fail block:
```go
extraction, err := b.extractor.ExtractText(ctx, sanitizeResult.Text, ctxRefs)
if err != nil {
    b.log.Error(ctx, "ingest", "msg", "ai extraction failed, continuing without", "error", err)
    pr.Extraction = &StepResult{
        Status: "failed",
        Detail: map[string]any{"error": err.Error()},
    }
    resultJSON, _ := json.Marshal(pr)
    raw := json.RawMessage(resultJSON)
    if _, err := b.rawInputBus.Update(ctx, ri, rawinputbus.UpdateRawInput{Result: &raw}); err != nil {
        b.log.Error(ctx, "ingest", "msg", "failed to save pipeline result", "error", err)
    }
    if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
        return IngestResult{}, fmt.Errorf("mark processed: %w", err)
    }
    return IngestResult{}, nil
}
```

**Step 5 (extraction) — on success path, after the existing log block (around line 550):**
```go
pr.Extraction = &StepResult{
    Status: "completed",
    Detail: map[string]any{
        "action_items": len(extraction.ActionItems),
        "events":       len(extraction.Events),
        "notes":        len(extraction.Notes),
    },
}
```

**Step 6 (context matching) — after the context matching logic completes (after auto-create context block, around line 603):**
```go
if matchedContextID != nil {
    pr.ContextMatch = &StepResult{
        Status: "completed",
        Detail: map[string]any{"context_id": matchedContextID.String()},
    }
} else {
    pr.ContextMatch = &StepResult{Status: "skipped"}
}
```

Note: place this AFTER all context matching logic (ID match, keyword fallback, auto-create) but BEFORE the low-confidence clarification block.

**Step 7 (task creation) — after the task creation loop (around line 657):**
```go
taskIDStrs := make([]string, len(createdTaskIDs))
for i, id := range createdTaskIDs {
    taskIDStrs[i] = id.String()
}
pr.Tasks = &StepResult{
    Status: "completed",
    Detail: map[string]any{"count": len(createdTaskIDs), "ids": taskIDStrs},
}
```

**After event creation loop (around line 701):**
```go
eventIDStrs := make([]string, len(createdEventIDs))
for i, id := range createdEventIDs {
    eventIDStrs[i] = id.String()
}
pr.Events = &StepResult{
    Status: "completed",
    Detail: map[string]any{"count": len(createdEventIDs), "ids": eventIDStrs},
}
```

**After note creation loop (around line 735):**
```go
noteIDStrs := make([]string, len(createdNoteIDs))
for i, id := range createdNoteIDs {
    noteIDStrs[i] = id.String()
}
pr.Notes = &StepResult{
    Status: "completed",
    Detail: map[string]any{"count": len(createdNoteIDs), "ids": noteIDStrs},
}
```

**Step 10 (mark processed) — before the MarkProcessed call (around line 813):**
Replace:
```go
if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
```
With:
```go
resultJSON, _ := json.Marshal(pr)
raw := json.RawMessage(resultJSON)
if _, err := b.rawInputBus.Update(ctx, ri, rawinputbus.UpdateRawInput{Result: &raw}); err != nil {
    b.log.Error(ctx, "ingest", "msg", "failed to save pipeline result", "error", err)
}
if _, err := b.rawInputBus.MarkProcessed(ctx, ri); err != nil {
```

- [ ] **Step 4: Verify it compiles**

```bash
cd /Users/casebrophy/personal/planner && go build ./...
```

- [ ] **Step 5: Run tests**

```bash
make test
```

- [ ] **Step 6: Commit**

```bash
git add business/domain/ingestbus/ingestbus.go
git commit -m "feat: instrument processTextInput with per-step pipeline result tracking"
```

---

### Task 4: Frontend — add types and store support

**Files:**
- Modify: `api/services/frontend/web/src/types/rawinput.ts`
- Modify: `api/services/frontend/web/src/stores/rawinputStore.ts`
- Modify: `api/services/frontend/web/src/services/rawinputService.ts`

- [ ] **Step 1: Add PipelineResult types**

Replace `api/services/frontend/web/src/types/rawinput.ts` with:

```ts
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

- [ ] **Step 2: Add fetchById to store**

In `api/services/frontend/web/src/stores/rawinputStore.ts`, add a `selectedItem` ref and `fetchById` action:

Add after `const statusFilter = ref<string | undefined>(undefined)`:
```ts
const selectedItem = ref<RawInput | null>(null)
```

Add after the `setPage` function:
```ts
async function fetchById(id: string) {
  loading.value = true
  try {
    selectedItem.value = await rawinputService.getById(id)
  } catch (e) {
    const msg = e instanceof Error ? e.message : 'Failed to fetch raw input'
    toast.error(msg)
    selectedItem.value = null
  } finally {
    loading.value = false
  }
}
```

Add `selectedItem` and `fetchById` to the return object.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/types/rawinput.ts api/services/frontend/web/src/stores/rawinputStore.ts
git commit -m "feat: add PipelineResult types and fetchById to rawinput store"
```

---

### Task 5: Frontend — RawInputDetailView drawer

**Files:**
- Create: `api/services/frontend/web/src/views/RawInputDetailView.vue`
- Modify: `api/services/frontend/web/src/views/RawInputQueueView.vue`
- Modify: `api/services/frontend/web/src/router/index.ts`

- [ ] **Step 1: Create RawInputDetailView.vue**

Create `api/services/frontend/web/src/views/RawInputDetailView.vue`:

```vue
<script setup lang="ts">
import { watchEffect, ref, computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useRawInputStore } from '@/stores/rawinputStore'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import type { StepResult } from '@/types/rawinput'

const route = useRoute()
const router = useRouter()
const store = useRawInputStore()
const rawContentExpanded = ref(false)

const id = computed(() => route.params.id as string)

watchEffect(async () => {
  if (id.value) {
    await store.fetchById(id.value)
  }
})

const item = computed(() => store.selectedItem)

const pipelineSteps = computed(() => {
  if (!item.value?.result) return []
  const r = item.value.result
  const steps: { name: string; label: string; result?: StepResult }[] = [
    { name: 'sanitize', label: 'Sanitize', result: r.sanitize },
    { name: 'extraction', label: 'AI Extraction', result: r.extraction },
    { name: 'contextMatch', label: 'Context Match', result: r.contextMatch },
    { name: 'tasks', label: 'Task Creation', result: r.tasks },
    { name: 'events', label: 'Event Creation', result: r.events },
    { name: 'notes', label: 'Note Creation', result: r.notes },
  ]
  return steps
})

function statusIcon(status?: string): string {
  if (status === 'completed') return '\u2713'
  if (status === 'failed') return '\u2717'
  if (status === 'skipped') return '\u2014'
  return '\u00B7'
}

function statusColor(status?: string): string {
  if (status === 'completed') return 'text-green-600 dark:text-green-400'
  if (status === 'failed') return 'text-red-600 dark:text-red-400'
  return 'text-gray-400 dark:text-gray-500'
}

function formatDetail(detail?: Record<string, unknown>): string {
  if (!detail) return ''
  return Object.entries(detail)
    .map(([k, v]) => {
      if (Array.isArray(v)) return `${k}: ${v.length}`
      return `${k}: ${v}`
    })
    .join(' · ')
}

function formatDate(iso?: string): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleString()
}

function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    processing: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    processed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  }
  return map[status] ?? 'bg-gray-100 text-gray-800'
}

async function handleReprocess() {
  if (!item.value) return
  await store.reprocess(item.value.id)
  await store.fetchById(item.value.id)
}
</script>

<template>
  <div class="p-6 space-y-6">
    <LoadingSpinner v-if="store.loading && !item" />

    <template v-else-if="item">
      <!-- Header -->
      <div class="space-y-2">
        <div class="flex items-center gap-3">
          <span class="font-mono text-sm text-gray-500 dark:text-gray-400">
            {{ item.sourceType }}
          </span>
          <span
            :class="[
              'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
              statusBadgeClass(item.status),
            ]"
          >
            {{ item.status }}
          </span>
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400 space-y-0.5">
          <div>Created: {{ formatDate(item.createdAt) }}</div>
          <div v-if="item.processedAt">
            Processed: {{ formatDate(item.processedAt) }}
          </div>
          <div v-if="item.retryCount > 0">
            Retries: {{ item.retryCount }} / {{ item.maxRetries }}
          </div>
        </div>
      </div>

      <!-- Error -->
      <div
        v-if="item.error"
        class="p-3 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800"
      >
        <p class="text-sm font-medium text-red-800 dark:text-red-200">
          Error
        </p>
        <p class="text-sm text-red-600 dark:text-red-300 mt-1 break-words">
          {{ item.error }}
        </p>
      </div>

      <!-- Pipeline Steps -->
      <div v-if="pipelineSteps.length > 0">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Pipeline Steps
        </h3>
        <div class="space-y-1">
          <div
            v-for="step in pipelineSteps"
            :key="step.name"
            class="flex items-start gap-2 py-1.5 px-2 rounded text-sm"
          >
            <span :class="['font-mono text-base leading-5 w-5 text-center shrink-0', statusColor(step.result?.status)]">
              {{ statusIcon(step.result?.status) }}
            </span>
            <div class="min-w-0">
              <span class="text-gray-800 dark:text-gray-200">{{ step.label }}</span>
              <span
                v-if="step.result?.detail"
                class="ml-2 text-xs text-gray-500 dark:text-gray-400"
              >
                {{ formatDetail(step.result.detail) }}
              </span>
              <p
                v-if="step.result?.status === 'failed' && step.result.detail?.error"
                class="text-xs text-red-500 mt-0.5 break-words"
              >
                {{ step.result.detail.error }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="item.status === 'processed' || item.status === 'failed'">
        <p class="text-sm text-gray-400 dark:text-gray-500 italic">
          No pipeline result recorded (processed before tracking was added).
        </p>
      </div>

      <!-- Raw Content -->
      <div>
        <button
          class="text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 flex items-center gap-1"
          @click="rawContentExpanded = !rawContentExpanded"
        >
          <span class="text-xs">{{ rawContentExpanded ? '\u25BC' : '\u25B6' }}</span>
          Raw Content
        </button>
        <pre
          v-if="rawContentExpanded"
          class="mt-2 p-3 rounded-lg bg-gray-50 dark:bg-gray-800 text-xs text-gray-700 dark:text-gray-300 overflow-x-auto whitespace-pre-wrap break-words max-h-64 overflow-y-auto"
        >{{ item.rawContent }}</pre>
      </div>

      <!-- Actions -->
      <div class="pt-2 border-t border-gray-200 dark:border-gray-700">
        <button
          v-if="item.status === 'failed' || item.status === 'processed'"
          class="text-sm px-3 py-1.5 rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
          :disabled="store.loading"
          @click="handleReprocess"
        >
          Reprocess
        </button>
      </div>
    </template>

    <div v-else class="text-center py-12 text-gray-400">
      Raw input not found.
    </div>
  </div>
</template>
```

- [ ] **Step 2: Update RawInputQueueView to support drawer**

In `api/services/frontend/web/src/views/RawInputQueueView.vue`:

Add imports at the top of `<script setup>`:
```ts
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'

const route = useRoute()
const router = useRouter()
const drawerOpen = computed(() => !!route.params.id)

function openDetail(id: string) {
  router.push({ name: 'rawinput-detail', params: { id } })
}

function closeDrawer() {
  router.push({ name: 'ingest-queue' })
}
```

Update the existing `onMounted`/`onUnmounted` import — change `import { onMounted, onUnmounted } from 'vue'` to `import { onMounted, onUnmounted, computed } from 'vue'` (merge with new imports).

Add click handler to each table row — change the `<tr>` inside tbody:
```html
<tr
  v-for="item in store.items"
  :key="item.id"
  class="bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800 cursor-pointer"
  @click="openDetail(item.id)"
>
```

Add the drawer at the bottom of the template, before the closing `</div>`:
```html
<DrawerPanel
  :open="drawerOpen"
  title="Raw Input Detail"
  @close="closeDrawer"
>
  <router-view />
</DrawerPanel>
```

- [ ] **Step 3: Add route**

In `api/services/frontend/web/src/router/index.ts`:

Add lazy import:
```ts
const RawInputDetailView = () => import('@/views/RawInputDetailView.vue')
```

Change the ingest-queue route from:
```ts
{ path: '/ingest-queue', name: 'ingest-queue', component: RawInputQueueView },
```
To:
```ts
{
  path: '/ingest-queue',
  name: 'ingest-queue',
  component: RawInputQueueView,
  children: [{ path: ':id', name: 'rawinput-detail', component: RawInputDetailView, props: true }],
},
```

- [ ] **Step 4: Verify frontend builds**

```bash
make frontend-build
```

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/views/RawInputDetailView.vue api/services/frontend/web/src/views/RawInputQueueView.vue api/services/frontend/web/src/router/index.ts
git commit -m "feat: add RawInputDetailView drawer with pipeline step tracking"
```

---

### Task 6: Frontend — tests for RawInputDetailView

**Files:**
- Create: `api/services/frontend/web/src/__tests__/views/RawInputDetailView.test.ts`

- [ ] **Step 1: Write tests**

Check existing test patterns first — look at an existing detail view test (e.g., `api/services/frontend/web/src/__tests__/views/TaskDetailView.test.ts`) for the test setup pattern (mocking router, stores, services).

Create `api/services/frontend/web/src/__tests__/views/RawInputDetailView.test.ts` with tests covering:

1. **Shows loading state** — renders LoadingSpinner when loading and no item
2. **Shows raw input header** — source type, status badge, timestamps
3. **Shows pipeline steps** — renders all 6 steps with correct status icons
4. **Shows failed step detail** — extraction failure shows error message
5. **Shows empty state for old items** — items with no result show "No pipeline result recorded" message
6. **Raw content toggle** — collapsed by default, expands on click
7. **Reprocess button** — visible for failed/processed items, calls store.reprocess

Base the test structure on the existing patterns in the test directory. Mock `useRoute` to return `params: { id: 'test-id' }`, mock the store's `fetchById` and `selectedItem`.

- [ ] **Step 2: Run tests**

```bash
make frontend-test
```

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/__tests__/views/RawInputDetailView.test.ts
git commit -m "test: add RawInputDetailView tests for pipeline step display"
```
