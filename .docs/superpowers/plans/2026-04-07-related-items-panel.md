# Related Items "In Same Context" Sub-list Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an "In same context" sub-list to the existing Related Items panel in TaskDetailView and NoteDetailView, showing tasks and notes that share the same `context_id` as the current entity.

**Architecture:** Each detail view gains a `useRelatedByContext` composable that takes a `contextId` and the current entity's type+id, then makes dedicated API calls to fetch tasks and notes filtered by that context (excluding self). This avoids mutating the shared CRUD stores. The existing "Also related" (explicit entity links) section stays unchanged.

**Tech Stack:** Vue 3, TypeScript, Pinia, Vitest

---

## File Structure

| Action | File | Responsibility |
|--------|------|----------------|
| CREATE | `web/src/composables/useRelatedByContext.ts` | Fetches tasks+notes sharing a contextId via the API services directly; returns reactive lists excluding the current entity |
| MODIFY | `web/src/views/TaskDetailView.vue` | Import composable, wire "In same context" sub-list above explicit links |
| MODIFY | `web/src/views/NoteDetailView.vue` | Same as TaskDetailView |
| CREATE | `web/src/__tests__/composables/useRelatedByContext.test.ts` | Unit tests for the composable |
| MODIFY | `web/src/__tests__/views/TaskDetailView.test.ts` | Tests for "In same context" section rendering |
| CREATE | `web/src/__tests__/views/NoteDetailView.test.ts` | Tests for NoteDetailView (no test file exists yet) |

All paths are relative to `api/services/frontend/`.

---

### Task 1: Create useRelatedByContext composable — tests

**Files:**
- Create: `api/services/frontend/web/src/__tests__/composables/useRelatedByContext.test.ts`

- [ ] **Step 1: Write the failing tests**

```typescript
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { useRelatedByContext } from '@/composables/useRelatedByContext'

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  },
}))

vi.mock('@/services/noteService', () => ({
  noteService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  },
}))

import { taskService } from '@/services/taskService'
import { noteService } from '@/services/noteService'

describe('useRelatedByContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns empty arrays when contextId is undefined', async () => {
    const { tasks, notes } = useRelatedByContext(ref(undefined), 'task', 'task-1')
    await nextTick()
    expect(tasks.value).toEqual([])
    expect(notes.value).toEqual([])
    expect(taskService.list).not.toHaveBeenCalled()
    expect(noteService.list).not.toHaveBeenCalled()
  })

  it('fetches tasks and notes when contextId is set', async () => {
    const mockTasks = [
      { id: 'task-2', title: 'Other task', contextId: 'ctx-1' },
      { id: 'task-1', title: 'Self', contextId: 'ctx-1' },
    ]
    const mockNotes = [
      { id: 'note-1', content: 'A note', contextId: 'ctx-1' },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 2 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 1 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'task', 'task-1')

    // Wait for both API calls to resolve
    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    // Self is excluded from tasks
    expect(tasks.value).toEqual([{ id: 'task-2', title: 'Other task', contextId: 'ctx-1' }])
    expect(notes.value).toEqual(mockNotes)
  })

  it('excludes self from notes when entityType is note', async () => {
    const mockTasks = [{ id: 'task-1', title: 'A task', contextId: 'ctx-1' }]
    const mockNotes = [
      { id: 'note-1', content: 'Self note', contextId: 'ctx-1' },
      { id: 'note-2', content: 'Other note', contextId: 'ctx-1' },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 1 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 2 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'note', 'note-1')

    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    expect(tasks.value).toEqual(mockTasks)
    // Self excluded from notes
    expect(notes.value).toEqual([{ id: 'note-2', content: 'Other note', contextId: 'ctx-1' }])
  })

  it('refetches when contextId changes', async () => {
    vi.mocked(taskService.list).mockResolvedValue({ items: [], total: 0 })
    vi.mocked(noteService.list).mockResolvedValue({ items: [], total: 0 })

    const contextId = ref<string | undefined>('ctx-1')
    const { loading } = useRelatedByContext(contextId, 'task', 'task-1')

    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })
    expect(taskService.list).toHaveBeenCalledTimes(1)

    contextId.value = 'ctx-2'
    await nextTick()
    await vi.waitFor(() => {
      expect(taskService.list).toHaveBeenCalledTimes(2)
    })
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd api/services/frontend && npx vitest run src/__tests__/composables/useRelatedByContext.test.ts`
Expected: FAIL — module `@/composables/useRelatedByContext` not found

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/__tests__/composables/useRelatedByContext.test.ts
git commit -m "test: add failing tests for useRelatedByContext composable"
```

---

### Task 2: Implement useRelatedByContext composable

**Files:**
- Create: `api/services/frontend/web/src/composables/useRelatedByContext.ts`

- [ ] **Step 1: Write minimal implementation**

```typescript
import { ref, watch, type Ref } from 'vue'
import { taskService } from '@/services/taskService'
import { noteService } from '@/services/noteService'
import type { Task, Note } from '@/types'

export function useRelatedByContext(
  contextId: Ref<string | undefined>,
  entityType: 'task' | 'note',
  entityId: string,
) {
  const tasks = ref<Task[]>([])
  const notes = ref<Note[]>([])
  const loading = ref(false)

  async function fetch() {
    const cid = contextId.value
    if (!cid) {
      tasks.value = []
      notes.value = []
      return
    }

    loading.value = true
    try {
      const [taskResult, noteResult] = await Promise.all([
        taskService.list({ page: 1, rows: 20, orderBy: 'created_at', filter: { contextId: cid } }),
        noteService.list({ page: 1, rows: 20, orderBy: 'created_at', filter: { contextId: cid } }),
      ])

      tasks.value = taskResult.items.filter(t => !(entityType === 'task' && t.id === entityId))
      notes.value = noteResult.items.filter(n => !(entityType === 'note' && n.id === entityId))
    } finally {
      loading.value = false
    }
  }

  watch(contextId, fetch, { immediate: true })

  return { tasks, notes, loading }
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `cd api/services/frontend && npx vitest run src/__tests__/composables/useRelatedByContext.test.ts`
Expected: All 4 tests PASS

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/composables/useRelatedByContext.ts
git commit -m "feat: add useRelatedByContext composable for same-context items"
```

---

### Task 3: Add "In same context" sub-list to TaskDetailView — tests

**Files:**
- Modify: `api/services/frontend/web/src/__tests__/views/TaskDetailView.test.ts`

- [ ] **Step 1: Add mock for useRelatedByContext and new tests**

Add this mock at the top of the file, after the existing mocks (after line 58):

```typescript
const mockRelatedTasks = { value: [] as any[] }
const mockRelatedNotes = { value: [] as any[] }

vi.mock('@/composables/useRelatedByContext', () => ({
  useRelatedByContext: vi.fn(() => ({
    tasks: mockRelatedTasks,
    notes: mockRelatedNotes,
    loading: { value: false },
  })),
}))
```

Add these test cases inside the `describe('TaskDetailView')` block, at the end (after the last `it` block):

```typescript
  it('does not render "In same context" section when task has no contextId', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('In same context')
    wrapper.unmount()
  })

  it('renders "In same context" section when related items exist', async () => {
    // Override useTaskDetail to return a task with contextId
    const { useTaskDetail } = await import('@/composables/useTaskDetail')
    vi.mocked(useTaskDetail).mockReturnValueOnce({
      task: { value: makeTask({ id: 'test-task-1', contextId: 'ctx-1' }) },
      tags: [],
      loading: false,
      update: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      addTag: vi.fn().mockResolvedValue(undefined),
      removeTag: vi.fn().mockResolvedValue(undefined),
    } as any)

    mockRelatedTasks.value = [makeTask({ id: 'task-2', title: 'Related task', contextId: 'ctx-1' })]
    mockRelatedNotes.value = [makeNote({ id: 'note-1', content: 'Related note', contextId: 'ctx-1' })]

    const { wrapper } = await mountView('test-task-1')
    await flushPromises()

    expect(wrapper.text()).toContain('In same context')
    expect(wrapper.text()).toContain('Related task')
    expect(wrapper.text()).toContain('Related note')

    mockRelatedTasks.value = []
    mockRelatedNotes.value = []
    wrapper.unmount()
  })
```

- [ ] **Step 2: Run tests to verify the new tests fail**

Run: `cd api/services/frontend && npx vitest run src/__tests__/views/TaskDetailView.test.ts`
Expected: The "renders In same context" test FAILs (text not found). The "does not render" test may PASS already.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/__tests__/views/TaskDetailView.test.ts
git commit -m "test: add failing tests for In same context section in TaskDetailView"
```

---

### Task 4: Add "In same context" sub-list to TaskDetailView — implementation

**Files:**
- Modify: `api/services/frontend/web/src/views/TaskDetailView.vue`

- [ ] **Step 1: Add import and composable call**

In `<script setup>` (line 7 area), add the import:

```typescript
import { useRelatedByContext } from '@/composables/useRelatedByContext'
```

After `const entityLinkStore = useEntityLinkStore()` (line 28), add:

```typescript
const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => task.value?.contextId),
  'task',
  taskId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)
```

- [ ] **Step 2: Add template section**

Inside the Related Items panel `<div class="mt-6">`, after the `<h3>` heading (after line 284) and before the `explicitLinks` section (before line 286), add:

```html
          <!-- In same context -->
          <div
            v-if="hasSameContextItems"
            class="mb-4"
          >
            <h4 class="text-xs font-medium text-gray-500 mb-2">
              In same context
            </h4>
            <div class="flex flex-col gap-2">
              <router-link
                v-for="t in sameContextTasks"
                :key="t.id"
                :to="{ name: 'task-detail', params: { id: t.id } }"
                class="flex items-center bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300 hover:bg-gray-600 transition-colors"
              >
                <span class="text-gray-500 mr-2 text-xs">task</span>
                {{ t.title }}
              </router-link>
              <router-link
                v-for="n in sameContextNotes"
                :key="n.id"
                :to="{ name: 'note-detail', params: { id: n.id } }"
                class="flex items-center bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300 hover:bg-gray-600 transition-colors"
              >
                <span class="text-gray-500 mr-2 text-xs">note</span>
                {{ n.content.slice(0, 80) }}{{ n.content.length > 80 ? '...' : '' }}
              </router-link>
            </div>
          </div>

          <!-- Also related (explicit links) -->
```

Also rename the existing `explicitLinks` section header. After the "Also related" comment, wrap the existing `v-if="explicitLinks.length > 0"` div with a sub-header:

```html
          <div
            v-if="explicitLinks.length > 0"
            class="mb-4"
          >
            <h4 class="text-xs font-medium text-gray-500 mb-2">
              Also related
            </h4>
            <div class="flex flex-col gap-2">
              <!-- existing link items here (move the v-for div inside) -->
            </div>
          </div>
```

- [ ] **Step 3: Run tests to verify they pass**

Run: `cd api/services/frontend && npx vitest run src/__tests__/views/TaskDetailView.test.ts`
Expected: All tests PASS

- [ ] **Step 4: Commit**

```bash
git add api/services/frontend/web/src/views/TaskDetailView.vue
git commit -m "feat: add In same context sub-list to TaskDetailView Related Items panel"
```

---

### Task 5: Add "In same context" sub-list to NoteDetailView — tests

**Files:**
- Create: `api/services/frontend/web/src/__tests__/views/NoteDetailView.test.ts`

- [ ] **Step 1: Write the test file**

```typescript
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import NoteDetailView from '@/views/NoteDetailView.vue'
import { makeNote, makeTask } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/stores/entityLinkStore', () => {
  const mockStore = {
    fetchLinks: vi.fn().mockResolvedValue(undefined),
    getLinks: vi.fn(() => []),
    createLink: vi.fn().mockResolvedValue(undefined),
    deleteLink: vi.fn().mockResolvedValue(undefined),
  }
  return {
    useEntityLinkStore: vi.fn(() => mockStore),
    entityLinkStore: mockStore,
  }
})

vi.mock('@/stores/tagStore', () => ({
  useTagStore: () => ({
    create: vi.fn().mockResolvedValue({ id: 'tag-1', name: 'test-tag' }),
  }),
}))

vi.mock('@/stores/contextStore', () => ({
  useContextStore: () => ({
    items: [],
    fetchList: vi.fn().mockResolvedValue(undefined),
  }),
}))

vi.mock('@/composables/useNoteDetail', () => ({
  useNoteDetail: vi.fn((noteId: string) => ({
    note: { value: makeNote({ id: noteId }) },
    tags: [],
    loading: false,
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    addTag: vi.fn().mockResolvedValue(undefined),
    removeTag: vi.fn().mockResolvedValue(undefined),
  })),
}))

const mockRelatedTasks = { value: [] as any[] }
const mockRelatedNotes = { value: [] as any[] }

vi.mock('@/composables/useRelatedByContext', () => ({
  useRelatedByContext: vi.fn(() => ({
    tasks: mockRelatedTasks,
    notes: mockRelatedNotes,
    loading: { value: false },
  })),
}))

async function mountView(noteId: string = 'test-note-1') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/notes', name: 'notes', component: { template: '<div />' } },
      { path: '/notes/:id', name: 'note-detail', component: NoteDetailView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
    ],
  })
  router.push(`/notes/${noteId}`)
  await router.isReady()

  return {
    wrapper: mount(NoteDetailView, {
      global: {
        plugins: [createPinia(), router],
        stubs: {
          LoadingSpinner: true,
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    }),
    router,
  }
}

describe('NoteDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockRelatedTasks.value = []
    mockRelatedNotes.value = []
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('renders Related Items section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Related Items')
    wrapper.unmount()
  })

  it('fetches entity links on mount', async () => {
    const { useEntityLinkStore } = await import('@/stores/entityLinkStore')
    const { wrapper } = await mountView('test-note-1')
    await flushPromises()

    const store = useEntityLinkStore()
    expect(store.fetchLinks).toHaveBeenCalledWith('note', 'test-note-1')
    wrapper.unmount()
  })

  it('does not render "In same context" section when note has no contextId', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('In same context')
    wrapper.unmount()
  })

  it('renders "In same context" section when related items exist', async () => {
    const { useNoteDetail } = await import('@/composables/useNoteDetail')
    vi.mocked(useNoteDetail).mockReturnValueOnce({
      note: { value: makeNote({ id: 'test-note-1', contextId: 'ctx-1' }) },
      tags: [],
      loading: false,
      update: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      addTag: vi.fn().mockResolvedValue(undefined),
      removeTag: vi.fn().mockResolvedValue(undefined),
    } as any)

    mockRelatedTasks.value = [makeTask({ id: 'task-1', title: 'Context task', contextId: 'ctx-1' })]
    mockRelatedNotes.value = [makeNote({ id: 'note-2', content: 'Context note', contextId: 'ctx-1' })]

    const { wrapper } = await mountView('test-note-1')
    await flushPromises()

    expect(wrapper.text()).toContain('In same context')
    expect(wrapper.text()).toContain('Context task')
    expect(wrapper.text()).toContain('Context note')

    wrapper.unmount()
  })

  it('shows "Link manually" button', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text().includes('Link manually'))).toBe(true)
    wrapper.unmount()
  })
})
```

- [ ] **Step 2: Run tests to verify failing state**

Run: `cd api/services/frontend && npx vitest run src/__tests__/views/NoteDetailView.test.ts`
Expected: The "In same context section when related items exist" test FAILs (the composable isn't wired in NoteDetailView yet)

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/__tests__/views/NoteDetailView.test.ts
git commit -m "test: add NoteDetailView tests including In same context section"
```

---

### Task 6: Add "In same context" sub-list to NoteDetailView — implementation

**Files:**
- Modify: `api/services/frontend/web/src/views/NoteDetailView.vue`

- [ ] **Step 1: Add import and composable call**

In `<script setup>` (line 7 area), add:

```typescript
import { useRelatedByContext } from '@/composables/useRelatedByContext'
```

After `const entityLinkStore = useEntityLinkStore()` (line 27), add:

```typescript
const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => note.value?.contextId),
  'note',
  noteId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)
```

- [ ] **Step 2: Add template section**

Inside the Related Items panel, after the `<h3>` heading (after line 213) and before the `explicitLinks` section, add the same "In same context" template block from Task 4, but using `note-detail` for the note links and `task-detail` for task links. Also add "Also related" sub-header around the explicit links section.

```html
          <!-- In same context -->
          <div
            v-if="hasSameContextItems"
            class="mb-4"
          >
            <h4 class="text-xs font-medium text-gray-500 mb-2">
              In same context
            </h4>
            <div class="flex flex-col gap-2">
              <router-link
                v-for="t in sameContextTasks"
                :key="t.id"
                :to="{ name: 'task-detail', params: { id: t.id } }"
                class="flex items-center bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300 hover:bg-gray-600 transition-colors"
              >
                <span class="text-gray-500 mr-2 text-xs">task</span>
                {{ t.title }}
              </router-link>
              <router-link
                v-for="n in sameContextNotes"
                :key="n.id"
                :to="{ name: 'note-detail', params: { id: n.id } }"
                class="flex items-center bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300 hover:bg-gray-600 transition-colors"
              >
                <span class="text-gray-500 mr-2 text-xs">note</span>
                {{ n.content.slice(0, 80) }}{{ n.content.length > 80 ? '...' : '' }}
              </router-link>
            </div>
          </div>

          <!-- Also related (explicit links) -->
```

Wrap the existing explicit links div with an "Also related" sub-header (same as Task 4).

- [ ] **Step 3: Run all tests**

Run: `cd api/services/frontend && npx vitest run src/__tests__/views/NoteDetailView.test.ts src/__tests__/views/TaskDetailView.test.ts src/__tests__/composables/useRelatedByContext.test.ts`
Expected: All tests PASS

- [ ] **Step 4: Run lint**

Run: `cd api/services/frontend && npx vue-tsc --noEmit && npx eslint web/src/views/TaskDetailView.vue web/src/views/NoteDetailView.vue web/src/composables/useRelatedByContext.ts`
Expected: No errors

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/views/NoteDetailView.vue
git commit -m "feat: add In same context sub-list to NoteDetailView Related Items panel"
```

---

### Task 7: Final verification and cleanup

- [ ] **Step 1: Run full frontend test suite**

Run: `cd api/services/frontend && npx vitest run`
Expected: All tests PASS

- [ ] **Step 2: Run lint on all changed files**

Run: `cd api/services/frontend && npm run lint`
Expected: No errors

- [ ] **Step 3: Final commit if any lint fixes were needed**

```bash
git add -A
git commit -m "chore: lint fixes for related items panel"
```
