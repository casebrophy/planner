# Frontend Clarification Queue + Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the clarification queue review deck as the centerpiece frontend feature, plus thread/observation panels on detail views and empty states.

**Architecture:** New `clarifications/` component domain with service, store, and composable following existing patterns. ClarificationCard renders per-kind UI, ClarificationSession manages the deck, ClarificationView is the route. ThreadPanel is a reusable shared component.

**Tech Stack:** Vue 3 (Composition API, `<script setup>`), TypeScript, Pinia, Tailwind CSS (dark theme), vue-router, date-fns

**Spec:** `docs/superpowers/specs/2026-03-20-phase3b-activation-and-clarification-ui-design.md`

**Verification:** `cd web && npm run lint && npm run build` after each task.

---

## File Map

### Files to create
| File | Purpose |
|------|---------|
| `web/src/types/clarification.ts` | ClarificationItem, NewClarification types + kind/status enums |
| `web/src/services/clarificationService.ts` | API client for clarification endpoints |
| `web/src/stores/clarificationStore.ts` | Pinia store for queue state |
| `web/src/composables/useClarification.ts` | Composable for ClarificationView |
| `web/src/components/clarifications/ClarificationCard.vue` | Single card renderer (per-kind) |
| `web/src/components/clarifications/ClarificationSession.vue` | Review deck manager |
| `web/src/views/ClarificationView.vue` | Route view |
| `web/src/services/threadService.ts` | API client for thread endpoints |
| `web/src/services/observationService.ts` | API client for observation endpoints |
| `web/src/components/shared/ThreadPanel.vue` | Reusable thread timeline |

### Files to modify
| File | Change |
|------|--------|
| `web/src/types/enums.ts` | Add ClarificationKind, ClarificationStatus enums |
| `web/src/types/index.ts` | Re-export clarification types |
| `web/src/router/index.ts` | Add `/clarifications` route |
| `web/src/components/layout/AppSidebar.vue` | Add Clarifications nav item with badge |
| `web/src/views/TaskDetailView.vue` | Embed ThreadPanel |
| `web/src/views/ContextDetailView.vue` | Embed ThreadPanel + observations |

---

## Task 1: Types & Enums

**Files:**
- Create: `web/src/types/clarification.ts`
- Modify: `web/src/types/enums.ts`
- Modify: `web/src/types/index.ts`

### Steps

- [ ] **Step 1: Add clarification enums**

In `web/src/types/enums.ts`, add:

```typescript
export const ClarificationKind = {
  ContextAssignment: 'context_assignment',
  StaleTask: 'stale_task',
  AmbiguousDeadline: 'ambiguous_deadline',
  NewContext: 'new_context',
  OverlappingContexts: 'overlapping_contexts',
  AmbiguousAction: 'ambiguous_action',
  VoiceReference: 'voice_reference',
  InactivityPrompt: 'inactivity_prompt',
  ContextDebrief: 'context_debrief',
} as const
export type ClarificationKind = (typeof ClarificationKind)[keyof typeof ClarificationKind]

export const ClarificationStatus = {
  Pending: 'pending',
  Snoozed: 'snoozed',
  Resolved: 'resolved',
  Dismissed: 'dismissed',
} as const
export type ClarificationStatus = (typeof ClarificationStatus)[keyof typeof ClarificationStatus]

export const ClarificationKindLabels: Record<ClarificationKind, string> = {
  [ClarificationKind.ContextAssignment]: 'Context Assignment',
  [ClarificationKind.StaleTask]: 'Stale Task',
  [ClarificationKind.AmbiguousDeadline]: 'Ambiguous Deadline',
  [ClarificationKind.NewContext]: 'New Context',
  [ClarificationKind.OverlappingContexts]: 'Overlapping Contexts',
  [ClarificationKind.AmbiguousAction]: 'Ambiguous Action',
  [ClarificationKind.VoiceReference]: 'Voice Reference',
  [ClarificationKind.InactivityPrompt]: 'Inactivity',
  [ClarificationKind.ContextDebrief]: 'Debrief',
}

export const ClarificationKindColors: Record<ClarificationKind, string> = {
  [ClarificationKind.ContextAssignment]: '#f59e0b',
  [ClarificationKind.StaleTask]: '#ef4444',
  [ClarificationKind.AmbiguousDeadline]: '#f97316',
  [ClarificationKind.NewContext]: '#8b5cf6',
  [ClarificationKind.OverlappingContexts]: '#6366f1',
  [ClarificationKind.AmbiguousAction]: '#f59e0b',
  [ClarificationKind.VoiceReference]: '#3b82f6',
  [ClarificationKind.InactivityPrompt]: '#ef4444',
  [ClarificationKind.ContextDebrief]: '#10b981',
}
```

- [ ] **Step 2: Create clarification type file**

Create `web/src/types/clarification.ts`:

```typescript
import type { ClarificationKind, ClarificationStatus } from './enums'

export interface ClarificationItem {
  id: string
  kind: ClarificationKind
  status: ClarificationStatus
  subjectType: string
  subjectId: string
  question: string
  claudeGuess?: Record<string, unknown>
  reasoning?: string
  answerOptions: Record<string, unknown>
  answer?: Record<string, unknown>
  priorityScore: number
  snoozedUntil?: string
  createdAt: string
  resolvedAt?: string
}

export interface ClarificationCountResponse {
  count: number
}
```

- [ ] **Step 3: Update index re-exports**

In `web/src/types/index.ts`, add:

```typescript
export type { ClarificationItem, ClarificationCountResponse } from './clarification'
```

- [ ] **Step 4: Run lint**

Run: `cd web && npm run lint`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/types/
git commit -m "feat: add clarification TypeScript types and enums"
```

---

## Task 2: Clarification Service

**Files:**
- Create: `web/src/services/clarificationService.ts`

### Steps

- [ ] **Step 1: Create the service**

Create `web/src/services/clarificationService.ts`:

```typescript
import { request } from './client'
import type { ClarificationItem, ClarificationCountResponse } from '@/types'
import type { QueryResult, ListParams } from '@/types/query'

interface ClarificationListParams extends ListParams {
  status?: string
}

interface ClarificationQueryResponse {
  items: ClarificationItem[]
  total: number
  page: number
  rowsPerPage: number
}

export const clarificationService = {
  async queryQueue(params: ClarificationListParams = {}): Promise<QueryResult<ClarificationItem>> {
    const queryParams: Record<string, string> = {}
    if (params.page) queryParams.page = String(params.page)
    if (params.rows) queryParams.rows_per_page = String(params.rows)
    if (params.status) queryParams.status = params.status
    if (params.orderBy) queryParams.orderBy = params.orderBy

    const res = await request<ClarificationQueryResponse>('/api/v1/clarifications', { params: queryParams })
    return { items: res.items, total: res.total, page: res.page, rowsPerPage: res.rowsPerPage }
  },

  async queryByID(id: string): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}`)
  },

  async countPending(): Promise<number> {
    const res = await request<ClarificationCountResponse>('/api/v1/clarifications/count')
    return res.count
  },

  async resolve(id: string, answer: Record<string, unknown>): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/resolve`, {
      method: 'POST',
      body: { answer },
    })
  },

  async snooze(id: string, hours: number = 24): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/snooze`, {
      method: 'POST',
      body: { hours },
    })
  },

  async dismiss(id: string): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/dismiss`, {
      method: 'POST',
    })
  },
}
```

- [ ] **Step 2: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/services/clarificationService.ts
git commit -m "feat: add clarification API service"
```

---

## Task 3: Clarification Store

**Files:**
- Create: `web/src/stores/clarificationStore.ts`

### Steps

- [ ] **Step 1: Create the Pinia store**

Create `web/src/stores/clarificationStore.ts`:

```typescript
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useToastStore } from './toastStore'
import { clarificationService } from '@/services/clarificationService'
import type { ClarificationItem } from '@/types'

export const useClarificationStore = defineStore('clarification', () => {
  const items = ref<ClarificationItem[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const currentIndex = ref(0)
  const pendingCount = ref(0)

  const toast = useToastStore()

  const currentItem = computed(() => items.value[currentIndex.value] ?? null)
  const hasNext = computed(() => currentIndex.value < items.value.length - 1)
  const isEmpty = computed(() => !loading.value && items.value.length === 0)
  const progress = computed(() => ({
    current: currentIndex.value + 1,
    total: items.value.length,
  }))

  async function fetchQueue(force = false) {
    loading.value = true
    error.value = null
    try {
      const result = await clarificationService.queryQueue({ status: 'pending', rows: 50 })
      items.value = result.items
      total.value = result.total
      currentIndex.value = 0
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch queue'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchPendingCount() {
    try {
      pendingCount.value = await clarificationService.countPending()
    } catch {
      // Silent fail for badge count
    }
  }

  async function resolve(id: string, answer: Record<string, unknown>) {
    try {
      await clarificationService.resolve(id, answer)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success('Resolved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to resolve'
      toast.error(msg)
      throw e
    }
  }

  async function snooze(id: string, hours: number = 24) {
    try {
      await clarificationService.snooze(id, hours)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success(`Snoozed for ${hours}h`)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to snooze'
      toast.error(msg)
      throw e
    }
  }

  async function dismiss(id: string) {
    try {
      await clarificationService.dismiss(id)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success('Dismissed')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to dismiss'
      toast.error(msg)
      throw e
    }
  }

  function removeAndAdvance(id: string) {
    const idx = items.value.findIndex((i) => i.id === id)
    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
      // Keep currentIndex valid
      if (currentIndex.value >= items.value.length) {
        currentIndex.value = Math.max(0, items.value.length - 1)
      }
    }
  }

  function goTo(index: number) {
    if (index >= 0 && index < items.value.length) {
      currentIndex.value = index
    }
  }

  return {
    items,
    total,
    loading,
    error,
    currentIndex,
    pendingCount,
    currentItem,
    hasNext,
    isEmpty,
    progress,
    fetchQueue,
    fetchPendingCount,
    resolve,
    snooze,
    dismiss,
    goTo,
  }
})
```

- [ ] **Step 2: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/stores/clarificationStore.ts
git commit -m "feat: add clarification Pinia store"
```

---

## Task 4: ClarificationCard Component

**Files:**
- Create: `web/src/components/clarifications/ClarificationCard.vue`

### Steps

- [ ] **Step 1: Create the component**

Create `web/src/components/clarifications/ClarificationCard.vue`:

```vue
<script setup lang="ts">
import { ref, computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { ClarificationKind, ClarificationKindLabels, ClarificationKindColors } from '@/types/enums'
import type { ClarificationItem } from '@/types'

const props = defineProps<{
  item: ClarificationItem
}>()

const emit = defineEmits<{
  resolve: [answer: Record<string, unknown>]
  snooze: [hours: number]
  dismiss: []
}>()

const debriefAnswer = ref('')

const kindLabel = computed(() => ClarificationKindLabels[props.item.kind] ?? props.item.kind)
const kindColor = computed(() => ClarificationKindColors[props.item.kind] ?? '#6b7280')
const age = computed(() => formatDistanceToNow(new Date(props.item.createdAt), { addSuffix: true }))

const options = computed(() => {
  if (!props.item.answerOptions) return {}
  return typeof props.item.answerOptions === 'string'
    ? JSON.parse(props.item.answerOptions)
    : props.item.answerOptions
})

function resolveWithValue(answer: Record<string, unknown>) {
  emit('resolve', answer)
}

function resolveDebrief() {
  if (debriefAnswer.value.trim()) {
    emit('resolve', { response: debriefAnswer.value.trim() })
  }
}
</script>

<template>
  <div class="bg-gray-800 rounded-xl p-6 border-l-4" :style="{ borderLeftColor: kindColor }">
    <!-- Header -->
    <div class="flex items-center gap-2 mb-3">
      <span
        class="px-2 py-0.5 rounded text-xs font-medium"
        :style="{ backgroundColor: kindColor + '22', color: kindColor }"
      >
        {{ kindLabel }}
      </span>
      <span class="text-gray-500 text-xs">{{ age }}</span>
    </div>

    <!-- Question -->
    <h3 class="text-lg font-semibold text-gray-100 mb-2">{{ item.question }}</h3>

    <!-- Reasoning (if present) -->
    <p v-if="item.reasoning" class="text-sm text-gray-400 mb-4">{{ item.reasoning }}</p>

    <!-- Kind-specific actions -->
    <div class="mt-4">
      <!-- Context Assignment -->
      <div v-if="item.kind === ClarificationKind.ContextAssignment" class="flex flex-col gap-2">
        <button
          v-if="options.suggested_context_id"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ context_id: options.suggested_context_id })"
        >
          Confirm suggested context
        </button>
        <button
          v-for="alt in (options.alternatives ?? [])"
          :key="alt"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
          @click="resolveWithValue({ context_id: alt })"
        >
          {{ alt }}
        </button>
      </div>

      <!-- Inactivity Prompt / Stale Task -->
      <div v-else-if="item.kind === ClarificationKind.InactivityPrompt || item.kind === ClarificationKind.StaleTask" class="flex gap-2">
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'extend' })"
        >
          Still active
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'note' })"
        >
          Add note
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'close' })"
        >
          Close
        </button>
      </div>

      <!-- Ambiguous Action -->
      <div v-else-if="item.kind === ClarificationKind.AmbiguousAction" class="flex flex-col gap-2">
        <button
          v-for="(interp, idx) in (options as unknown[])"
          :key="idx"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors text-left"
          @click="resolveWithValue({ selected: idx })"
        >
          {{ typeof interp === 'string' ? interp : JSON.stringify(interp) }}
        </button>
      </div>

      <!-- Ambiguous Deadline -->
      <div v-else-if="item.kind === ClarificationKind.AmbiguousDeadline" class="flex flex-col gap-2">
        <input
          type="datetime-local"
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @change="(e) => resolveWithValue({ due_date: new Date((e.target as HTMLInputElement).value).toISOString() })"
        />
      </div>

      <!-- New Context -->
      <div v-else-if="item.kind === ClarificationKind.NewContext" class="flex gap-2">
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'confirm' })"
        >
          Confirm
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'merge' })"
        >
          Merge
        </button>
      </div>

      <!-- Context Debrief -->
      <div v-else-if="item.kind === ClarificationKind.ContextDebrief" class="flex flex-col gap-2">
        <textarea
          v-model="debriefAnswer"
          rows="3"
          placeholder="Your answer..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        />
        <button
          :disabled="!debriefAnswer.trim()"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="resolveDebrief"
        >
          Submit
        </button>
      </div>

      <!-- Voice Reference -->
      <div v-else-if="item.kind === ClarificationKind.VoiceReference" class="flex flex-col gap-2">
        <input
          type="text"
          placeholder="Corrected reference..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @keyup.enter="(e) => resolveWithValue({ resolved_text: (e.target as HTMLInputElement).value })"
        />
      </div>

      <!-- Fallback -->
      <div v-else class="flex gap-2">
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ acknowledged: true })"
        >
          Acknowledge
        </button>
      </div>
    </div>

    <!-- Snooze / Dismiss -->
    <div class="flex gap-2 mt-3">
      <button
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
        @click="emit('snooze', 24)"
      >
        Snooze 24h
      </button>
      <button
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
        @click="emit('dismiss')"
      >
        Dismiss
      </button>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/components/clarifications/
git commit -m "feat: add ClarificationCard component with per-kind rendering"
```

---

## Task 5: ClarificationSession Component

**Files:**
- Create: `web/src/components/clarifications/ClarificationSession.vue`

### Steps

- [ ] **Step 1: Create the component**

Create `web/src/components/clarifications/ClarificationSession.vue`:

```vue
<script setup lang="ts">
import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useClarificationStore } from '@/stores/clarificationStore'
import ClarificationCard from './ClarificationCard.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'

const store = useClarificationStore()
const { currentItem, loading, isEmpty, progress, items } = storeToRefs(store)

// NOTE: Do NOT fetch here — the parent ClarificationView's useClarification composable
// already calls fetchQueue() on mount. Fetching here would cause a double-fetch.

async function handleResolve(answer: Record<string, unknown>) {
  if (currentItem.value) {
    await store.resolve(currentItem.value.id, answer)
  }
}

async function handleSnooze(hours: number) {
  if (currentItem.value) {
    await store.snooze(currentItem.value.id, hours)
  }
}

async function handleDismiss() {
  if (currentItem.value) {
    await store.dismiss(currentItem.value.id)
  }
}
</script>

<template>
  <div>
    <LoadingSpinner v-if="loading && items.length === 0" />

    <EmptyState
      v-else-if="isEmpty"
      title="All caught up"
      message="No pending clarifications. Nice work!"
    />

    <div v-else>
      <!-- Progress -->
      <div class="flex items-center justify-between mb-4">
        <span class="text-sm text-gray-500">
          {{ progress.current }} of {{ progress.total }}
        </span>
      </div>

      <!-- Current Card -->
      <Transition name="slide" mode="out-in">
        <ClarificationCard
          v-if="currentItem"
          :key="currentItem.id"
          :item="currentItem"
          @resolve="handleResolve"
          @snooze="handleSnooze"
          @dismiss="handleDismiss"
        />
      </Transition>

      <!-- Progress Dots -->
      <div v-if="items.length > 1" class="flex justify-center gap-2 mt-4">
        <button
          v-for="(item, idx) in items"
          :key="item.id"
          class="w-2.5 h-2.5 rounded-full transition-colors"
          :class="idx === store.currentIndex ? 'bg-amber-500' : 'bg-gray-700'"
          @click="store.goTo(idx)"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.slide-enter-active,
.slide-leave-active {
  transition: all 0.2s ease;
}
.slide-enter-from {
  opacity: 0;
  transform: translateX(20px);
}
.slide-leave-to {
  opacity: 0;
  transform: translateX(-20px);
}
</style>
```

- [ ] **Step 2: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/components/clarifications/ClarificationSession.vue
git commit -m "feat: add ClarificationSession review deck component"
```

---

## Task 6: ClarificationView + Routing + Sidebar

**Files:**
- Create: `web/src/views/ClarificationView.vue`
- Create: `web/src/composables/useClarification.ts`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/layout/AppSidebar.vue`

### Steps

- [ ] **Step 1: Create the composable**

Create `web/src/composables/useClarification.ts`:

```typescript
import { onMounted, onUnmounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useClarificationStore } from '@/stores/clarificationStore'

export function useClarification() {
  const store = useClarificationStore()
  const { items, total, loading, error, currentItem, isEmpty, progress, pendingCount } =
    storeToRefs(store)

  // Pending count polling is handled globally by AppSidebar (60s interval).
  // This composable only fetches the queue itself.
  onMounted(() => {
    store.fetchQueue()
  })

  return {
    items,
    total,
    loading,
    error,
    currentItem,
    isEmpty,
    progress,
    pendingCount,
    resolve: store.resolve,
    snooze: store.snooze,
    dismiss: store.dismiss,
    refresh: () => store.fetchQueue(true),
  }
}
```

- [ ] **Step 2: Create the view**

Create `web/src/views/ClarificationView.vue`:

```vue
<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import ClarificationSession from '@/components/clarifications/ClarificationSession.vue'
import { useClarification } from '@/composables/useClarification'

const { pendingCount, refresh } = useClarification()
</script>

<template>
  <div>
    <PageHeader title="Clarifications" :subtitle="`${pendingCount} pending`">
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <div class="p-6 max-w-2xl mx-auto">
      <ClarificationSession />
    </div>
  </div>
</template>
```

- [ ] **Step 3: Add route**

In `web/src/router/index.ts`, add the import and route:

```typescript
const ClarificationView = () => import('@/views/ClarificationView.vue')

// In routes array:
{ path: '/clarifications', name: 'clarifications', component: ClarificationView },
```

- [ ] **Step 4: Add sidebar nav item with badge**

In `web/src/components/layout/AppSidebar.vue`, add to the `navItems` array:

```typescript
{ name: 'Clarifications', path: '/clarifications', icon: 'alert-circle' },
```

Add a badge for pending count. Import `useClarificationStore` and poll `fetchPendingCount`:

```typescript
import { useClarificationStore } from '@/stores/clarificationStore'
import { onMounted, onUnmounted } from 'vue'

const clarificationStore = useClarificationStore()
let countInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  clarificationStore.fetchPendingCount()
  countInterval = setInterval(() => clarificationStore.fetchPendingCount(), 60000)
})

onUnmounted(() => {
  if (countInterval) clearInterval(countInterval)
})
```

In the template, next to the "Clarifications" nav item text, add:

```vue
<span
  v-if="item.name === 'Clarifications' && clarificationStore.pendingCount > 0"
  class="ml-auto bg-amber-500 text-gray-900 text-xs font-bold px-1.5 py-0.5 rounded-full"
>
  {{ clarificationStore.pendingCount }}
</span>
```

- [ ] **Step 5: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add web/src/views/ClarificationView.vue web/src/composables/useClarification.ts web/src/router/index.ts web/src/components/layout/AppSidebar.vue
git commit -m "feat: add clarification view, route, and sidebar badge"
```

---

## Task 7: Thread & Observation Services

**Files:**
- Create: `web/src/services/threadService.ts`
- Create: `web/src/services/observationService.ts`

### Steps

- [ ] **Step 1: Create thread service**

Create `web/src/services/threadService.ts`:

```typescript
import { request } from './client'

export interface ThreadEntry {
  id: string
  subjectType: string
  subjectId: string
  kind: string
  content: string
  source: string
  metadata?: Record<string, unknown>
  sourceId?: string
  sentiment?: string
  requiresAction: boolean
  createdAt: string
}

interface ThreadQueryResponse {
  items: ThreadEntry[]
  total: number
  page: number
  rowsPerPage: number
}

export const threadService = {
  async queryBySubject(subjectType: string, subjectId: string): Promise<ThreadEntry[]> {
    const res = await request<ThreadQueryResponse>(
      `/api/v1/threads/${subjectType}/${subjectId}`,
    )
    return res.items
  },
}
```

- [ ] **Step 2: Create observation service**

Create `web/src/services/observationService.ts`:

```typescript
import { request } from './client'

export interface Observation {
  id: string
  subjectType: string
  subjectId: string
  kind: string
  data: Record<string, unknown>
  source: string
  confidence: number
  weight: number
  createdAt: string
}

interface ObservationQueryResponse {
  items: Observation[]
  total: number
  page: number
  rowsPerPage: number
}

export const observationService = {
  async queryBySubject(subjectType: string, subjectId: string): Promise<Observation[]> {
    const res = await request<ObservationQueryResponse>(
      `/api/v1/observations/${subjectType}/${subjectId}`,
    )
    return res.items
  },
}
```

- [ ] **Step 3: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add web/src/services/threadService.ts web/src/services/observationService.ts
git commit -m "feat: add thread and observation API services"
```

---

## Task 8: ThreadPanel + Detail View Integration

**Files:**
- Create: `web/src/components/shared/ThreadPanel.vue`
- Modify: `web/src/views/TaskDetailView.vue`
- Modify: `web/src/views/ContextDetailView.vue`

### Steps

- [ ] **Step 1: Create ThreadPanel component**

Create `web/src/components/shared/ThreadPanel.vue`:

```vue
<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { threadService, type ThreadEntry } from '@/services/threadService'
import LoadingSpinner from './LoadingSpinner.vue'

const props = defineProps<{
  subjectType: string
  subjectId: string
}>()

const entries = ref<ThreadEntry[]>([])
const loading = ref(false)

const kindIcons: Record<string, string> = {
  note: 'N',
  status_change: 'S',
  update: 'U',
  decision: 'D',
}

const sourceColors: Record<string, string> = {
  user: '#3b82f6',
  claude: '#8b5cf6',
  system: '#6b7280',
  email: '#f59e0b',
}

async function load() {
  loading.value = true
  try {
    entries.value = await threadService.queryBySubject(props.subjectType, props.subjectId)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <h4 class="text-sm font-medium text-gray-300 mb-3">Activity</h4>

    <LoadingSpinner v-if="loading" size="sm" />

    <div v-else-if="entries.length === 0" class="text-sm text-gray-500">
      No activity yet.
    </div>

    <div v-else class="space-y-3">
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex gap-3"
      >
        <!-- Kind indicator -->
        <div
          class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0"
          :style="{ backgroundColor: (sourceColors[entry.source] ?? '#6b7280') + '22', color: sourceColors[entry.source] ?? '#6b7280' }"
        >
          {{ kindIcons[entry.kind] ?? '?' }}
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm text-gray-200">{{ entry.content }}</p>
          <p class="text-xs text-gray-500 mt-0.5">
            {{ entry.source }} · {{ formatDistanceToNow(new Date(entry.createdAt), { addSuffix: true }) }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Embed ThreadPanel in TaskDetailView**

In `web/src/views/TaskDetailView.vue`, import and add after the tags section:

```vue
<script setup>
// Add import:
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
</script>

<!-- In template, after tags section, inside the v-if="!editing" block: -->
<div class="mt-6">
  <ThreadPanel subject-type="task" :subject-id="taskId" />
</div>
```

- [ ] **Step 3: Embed ThreadPanel in ContextDetailView**

In `web/src/views/ContextDetailView.vue`, import and add:

```vue
<script setup>
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import { observationService, type Observation } from '@/services/observationService'

const observations = ref<Observation[]>([])

// In the load function or onMounted, add:
observationService.queryBySubject('context', contextId).then(obs => {
  observations.value = obs
})
</script>

<!-- In template, after events section: -->
<div class="mt-6">
  <ThreadPanel subject-type="context" :subject-id="contextId" />
</div>

<div v-if="observations.length > 0" class="mt-6">
  <h4 class="text-sm font-medium text-gray-300 mb-3">Observations</h4>
  <div class="space-y-2">
    <div v-for="obs in observations" :key="obs.id" class="bg-gray-800 rounded-lg p-3">
      <p class="text-sm text-gray-200">{{ typeof obs.data === 'object' ? JSON.stringify(obs.data) : obs.data }}</p>
      <p class="text-xs text-gray-500 mt-1">{{ obs.kind }} · {{ obs.source }}</p>
    </div>
  </div>
</div>
```

- [ ] **Step 4: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/components/shared/ThreadPanel.vue web/src/views/TaskDetailView.vue web/src/views/ContextDetailView.vue
git commit -m "feat: add ThreadPanel and embed in detail views with observations"
```

---

## Task 9: Empty States on Existing Views

**Files:**
- Modify: `web/src/views/TaskBoardView.vue`
- Modify: `web/src/views/ContextBoardView.vue`
- Modify: `web/src/views/DashboardView.vue`

### Steps

- [ ] **Step 1: Add EmptyState to TaskBoardView**

In `web/src/views/TaskBoardView.vue`, import `EmptyState` and add between the loading spinner and the grid:

```vue
<EmptyState
  v-else-if="isEmpty"
  title="No tasks yet"
  message="Create your first task to get started"
  action-label="New Task"
  @action="showCreateForm = true"
/>
```

The `isEmpty` computed should already exist from the `useTaskBoard` composable. If not, add:
```typescript
const isEmpty = computed(() => !loading.value && tasks.value.length === 0)
```

- [ ] **Step 2: Add EmptyState to ContextBoardView**

In `web/src/views/ContextBoardView.vue`, import `EmptyState` and add:

```vue
<EmptyState
  v-else-if="isEmpty"
  title="No contexts yet"
  message="Create your first context to organize your work"
  action-label="New Context"
  @action="showCreateForm = true"
/>
```

- [ ] **Step 3: Add EmptyState to DashboardView**

In `web/src/views/DashboardView.vue`, add an empty state when there are no tasks and no contexts:

```vue
<EmptyState
  v-if="taskCount === 0 && contextCount === 0 && !loading"
  title="Welcome to Planner"
  message="Start by creating a task or context"
/>
```

- [ ] **Step 4: Run lint & build**

Run: `cd web && npm run lint && npm run build`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/views/TaskBoardView.vue web/src/views/ContextBoardView.vue web/src/views/DashboardView.vue
git commit -m "feat: add empty states to board and dashboard views"
```

---

## Final Verification

- [ ] **Run full lint and build**

```bash
cd web && npm run lint && npm run build
```

Expected: All pass. The frontend clarification queue and polish are complete.
