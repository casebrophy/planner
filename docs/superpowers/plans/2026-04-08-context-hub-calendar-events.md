# Context Hub Calendar Events Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `context_events` "Events" collapsible in the context hub with a sidebar calendar events card (showing real `events` table data) plus a create drawer, and remove all context_events plumbing from the frontend.

**Architecture:** Remove `fetchEvents`/`addEvent` from `contextStore` and `contextService`, remove `EventTimeline`/`EventForm` components, add an `initialContextId` prop to the existing `CalendarEventForm`, then wire a sidebar events card + `DrawerPanel` into both project and area hub sidebars in `ContextDetailView`.

**Tech Stack:** Vue 3, TypeScript, Pinia, Vitest, @vue/test-utils

---

## File Map

| File | Change |
|------|--------|
| `src/components/calendar-events/CalendarEventForm.vue` | Add `initialContextId` prop; hide context picker when set |
| `src/stores/contextStore.ts` | Remove `events`, `eventsTotal`, `fetchEvents`, `addEvent` |
| `src/services/contextService.ts` | Remove `listEvents`, `addEvent` |
| `src/composables/useContextDetail.ts` | Remove `events`, `eventsTotal`, `addEvent` |
| `src/views/ContextDetailView.vue` | Remove Events collapsible; add sidebar Events card + create drawer |
| `src/components/events/EventForm.vue` | **Delete** |
| `src/components/events/EventTimeline.vue` | **Delete** |
| `src/components/events/EventTimelineItem.vue` | **Delete** |
| `src/__tests__/components/calendar-events/CalendarEventForm.test.ts` | Add tests for `initialContextId` prop |
| `src/__tests__/stores/contextStore.test.ts` | Remove `fetchEvents`/`addEvent` tests; remove from mock |
| `src/__tests__/services/contextService.test.ts` | Remove `addEvent` test |
| `src/__tests__/composables/useContextDetail.test.ts` | Remove events/addEvent assertions; update mock |
| `src/__tests__/views/ContextDetailView.test.ts` | Update `useContextDetail` mock shape |
| `src/__tests__/components/events/EventForm.test.ts` | **Delete** |
| `src/__tests__/components/events/EventTimeline.test.ts` | **Delete** |
| 14 other test files | Remove `addEvent: vi.fn()` from contextStore mock |

---

### Task 1: Add `initialContextId` prop to CalendarEventForm

**Files:**
- Modify: `src/components/calendar-events/CalendarEventForm.vue`
- Create: `src/__tests__/components/calendar-events/CalendarEventForm.test.ts`

- [ ] **Step 1: Write the failing tests**

Create `src/__tests__/components/calendar-events/CalendarEventForm.test.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CalendarEventForm from '@/components/calendar-events/CalendarEventForm.vue'
import type { NewCalendarEvent } from '@/types/calendarEvent'

vi.mock('@/stores/contextStore', () => ({
  useContextStore: () => ({
    items: [],
    fetchList: vi.fn(),
  }),
}))

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('CalendarEventForm — initialContextId', () => {
  it('hides context picker when initialContextId is provided', () => {
    const wrapper = mount(CalendarEventForm, {
      props: { initialContextId: 'ctx-123' },
    })
    // The context <select> should not be rendered
    const labels = wrapper.findAll('label')
    const contextLabel = labels.find((l) => l.text() === 'Context')
    expect(contextLabel).toBeUndefined()
  })

  it('shows context picker when initialContextId is not provided', () => {
    const wrapper = mount(CalendarEventForm, {
      props: {},
    })
    const labels = wrapper.findAll('label')
    const contextLabel = labels.find((l) => l.text() === 'Context')
    expect(contextLabel).toBeDefined()
  })

  it('pre-populates contextId in emitted save payload when initialContextId is set', async () => {
    const wrapper = mount(CalendarEventForm, {
      props: { initialContextId: 'ctx-123' },
    })
    await wrapper.find('input[type="text"]').setValue('Team standup')
    const datetimeInputs = wrapper.findAll('input[type="datetime-local"]')
    await datetimeInputs[0]!.setValue('2026-04-10T10:00')
    await datetimeInputs[1]!.setValue('2026-04-10T11:00')
    await wrapper.find('form').trigger('submit')

    const emitted = wrapper.emitted('save')
    expect(emitted).toHaveLength(1)
    expect((emitted![0]![0] as NewCalendarEvent).contextId).toBe('ctx-123')
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd api/services/frontend/web && npx vitest run src/__tests__/components/calendar-events/CalendarEventForm.test.ts
```

Expected: FAIL — `initialContextId` prop not defined, context picker always shown.

- [ ] **Step 3: Add `initialContextId` prop to CalendarEventForm**

In `src/components/calendar-events/CalendarEventForm.vue`, update the `<script setup>`:

Change the props definition from:
```ts
const props = defineProps<{
  event?: CalendarEvent | null
}>()
```

To:
```ts
const props = defineProps<{
  event?: CalendarEvent | null
  initialContextId?: string
}>()
```

Change the `contextId` ref initialization from:
```ts
const contextId = ref(props.event?.contextId ?? '')
```

To:
```ts
const contextId = ref(props.event?.contextId ?? props.initialContextId ?? '')
```

In the template, wrap the context picker `<div>` with `v-if`:

Change:
```html
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Context</label>
        <select
          v-model="contextId"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
          <option value="">
            No context
          </option>
          <option
            v-for="ctx in contextStore.items"
            :key="ctx.id"
            :value="ctx.id"
          >
            {{ ctx.title }}
          </option>
        </select>
      </div>
```

To:
```html
      <div v-if="!initialContextId">
        <label class="block text-sm font-medium text-gray-300 mb-1">Context</label>
        <select
          v-model="contextId"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
          <option value="">
            No context
          </option>
          <option
            v-for="ctx in contextStore.items"
            :key="ctx.id"
            :value="ctx.id"
          >
            {{ ctx.title }}
          </option>
        </select>
      </div>
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd api/services/frontend/web && npx vitest run src/__tests__/components/calendar-events/CalendarEventForm.test.ts
```

Expected: all 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/components/calendar-events/CalendarEventForm.vue \
        api/services/frontend/web/src/__tests__/components/calendar-events/CalendarEventForm.test.ts
git commit -m "feat: add initialContextId prop to CalendarEventForm"
```

---

### Task 2: Remove context_events plumbing from store, service, and composable

**Files:**
- Modify: `src/stores/contextStore.ts`
- Modify: `src/services/contextService.ts`
- Modify: `src/composables/useContextDetail.ts`
- Modify: `src/__tests__/stores/contextStore.test.ts`
- Modify: `src/__tests__/services/contextService.test.ts`
- Modify: `src/__tests__/composables/useContextDetail.test.ts`

- [ ] **Step 1: Update `contextStore.ts`**

Replace the entire file content with:

```ts
import { defineStore } from 'pinia'
import { computed } from 'vue'
import { contextService } from '@/services/contextService'
import { createCRUDStore } from './createCRUDStore'
import type { Context, NewContext, UpdateContext, ContextFilter } from '@/types'
import { ContextStatus, ContextKind } from '@/types'

export const useContextStore = defineStore('context', () => {
  const crud = createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>({
    name: 'context',
    service: contextService,
    defaultOrderBy: 'last_event',
    defaultRowsPerPage: 50,
  })

  const contextsByStatus = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextStatus.Active]: [],
      [ContextStatus.Paused]: [],
      [ContextStatus.Closed]: [],
    }
    for (const ctx of crud.items.value) {
      groups[ctx.status]?.push(ctx)
    }
    return groups
  })

  const contextsByKind = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextKind.Project]: [],
      [ContextKind.Area]: [],
    }
    for (const ctx of crud.items.value) {
      const bucket = groups[ctx.kind]
      if (bucket) {
        bucket.push(ctx)
      } else {
        groups[ContextKind.Project]!.push(ctx)
      }
    }
    return groups
  })

  const contextById = computed(() => (id: string): Context | undefined => {
    return crud.items.value.find((c) => c.id === id)
  })

  const activeCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Active).length)
  const pausedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Paused).length)
  const closedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Closed).length)

  return {
    ...crud,
    contextsByStatus,
    contextsByKind,
    contextById,
    activeCount,
    pausedCount,
    closedCount,
  }
})
```

- [ ] **Step 2: Update `contextService.ts`**

Replace the file with:

```ts
import { createCRUDService } from './createCRUDService'
import type {
  Context,
  NewContext,
  UpdateContext,
  ContextFilter,
} from '@/types'

const crud = createCRUDService<Context, NewContext, UpdateContext, ContextFilter>({
  basePath: '/api/v1/contexts',
  mapFilter: (f) => ({
    status: f.status,
    kind: f.kind,
    title: f.title,
    parent_context_id: f.parentContextId,
  }),
})

export const contextService = {
  ...crud,
}
```

- [ ] **Step 3: Update `useContextDetail.ts`**

Replace the file with:

```ts
import { onMounted, computed } from 'vue'
import { useContextStore } from '@/stores/contextStore'
import { useTagStore } from '@/stores/tagStore'
import { useTaskStore } from '@/stores/taskStore'
import { storeToRefs } from 'pinia'
import type { UpdateContext } from '@/types'

export function useContextDetail(contextId: string) {
  const contextStore = useContextStore()
  const tagStore = useTagStore()
  const taskStore = useTaskStore()
  const { currentItem: currentContext, loading } = storeToRefs(contextStore)

  const tags = computed(() => tagStore.contextTags[contextId] ?? [])
  const linkedTasks = computed(() => taskStore.items.filter((t) => t.contextId === contextId))

  async function load() {
    await Promise.all([
      contextStore.fetchById(contextId),
      tagStore.fetchTagsForContext(contextId),
      taskStore.fetchList(true),
    ])
  }

  async function update(data: UpdateContext) {
    return contextStore.update(contextId, data)
  }

  async function remove() {
    return contextStore.remove(contextId)
  }

  async function addTag(tagId: string) {
    return tagStore.addTagToContext(contextId, tagId)
  }

  async function removeTag(tagId: string) {
    return tagStore.removeTagFromContext(contextId, tagId)
  }

  onMounted(load)

  return {
    context: currentContext,
    tags,
    linkedTasks,
    loading,
    update,
    remove,
    addTag,
    removeTag,
    reload: load,
  }
}
```

- [ ] **Step 4: Update `contextStore.test.ts` — remove fetchEvents/addEvent tests and update mock**

In `src/__tests__/stores/contextStore.test.ts`:

Remove the `listEvents: vi.fn()` and `addEvent: vi.fn()` entries from the `vi.mock('@/services/contextService', ...)` block:

```ts
vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))
```

Remove the `vi.mock('@/stores/toastStore', ...)` block entirely (the store no longer imports it).

Remove the entire `describe('fetchEvents', ...)` block (lines ~84–115).

Remove the entire `describe('addEvent', ...)` block and everything after it that covers `addEvent`.

Remove the imports `ContextEvent` and `NewEvent` from the top of the file.

- [ ] **Step 5: Update `contextService.test.ts` — remove addEvent test**

In `src/__tests__/services/contextService.test.ts`:

Remove the entire `describe('addEvent', ...)` block (~5 lines).

Remove the `listEvents` and `addEvent` mock setup if present.

Remove unused imports: `ContextEvent`, `NewEvent`.

- [ ] **Step 6: Update `useContextDetail.test.ts` — remove events/addEvent tests**

Replace `src/__tests__/composables/useContextDetail.test.ts` with:

```ts
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useContextDetail } from '@/composables/useContextDetail'
import { makeContext, makeTag, makeTask, makeQueryResult } from '../helpers/testFactories'
import { contextService } from '@/services/contextService'
import { tagService } from '@/services/tagService'
import { taskService } from '@/services/taskService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/services/tagService', () => ({
  tagService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getByTask: vi.fn(),
    addToTask: vi.fn(),
    removeFromTask: vi.fn(),
    getByContext: vi.fn(),
    addToContext: vi.fn(),
    removeFromContext: vi.fn(),
  },
}))

function withSetup<T>(composable: () => T) {
  let result!: T
  const wrapper = mount(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
    { global: { plugins: [createPinia()] } },
  )
  return { result, wrapper }
}

describe('useContextDetail', () => {
  let wrapper: ReturnType<typeof mount>

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('loads context, tags, and tasks on mount', async () => {
    const ctx = makeContext()
    const tag = makeTag()

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(tagService.getByContext).mockResolvedValue([tag])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(contextService.getById).toHaveBeenCalledWith(ctx.id)
    expect(tagService.getByContext).toHaveBeenCalledWith(ctx.id)
    expect(setup.result.context.value).toEqual(ctx)
    expect(setup.result.tags.value).toEqual([tag])
  })

  it('linkedTasks filters tasks by contextId', async () => {
    const ctx = makeContext()
    const linkedTask = makeTask({ contextId: ctx.id })
    const otherTask = makeTask({ contextId: 'other-ctx' })

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([linkedTask, otherTask]))

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(setup.result.linkedTasks.value).toHaveLength(1)
    expect(setup.result.linkedTasks.value[0]!.id).toBe(linkedTask.id)
  })

  it('update delegates to contextStore.update with contextId', async () => {
    const ctx = makeContext()
    const updatedCtx = { ...ctx, title: 'Updated' }

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.update).mockResolvedValue(updatedCtx)

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.update({ title: 'Updated' })

    expect(contextService.update).toHaveBeenCalledWith(ctx.id, { title: 'Updated' })
  })
})
```

- [ ] **Step 7: Run the affected tests**

```bash
cd api/services/frontend/web && npx vitest run \
  src/__tests__/stores/contextStore.test.ts \
  src/__tests__/services/contextService.test.ts \
  src/__tests__/composables/useContextDetail.test.ts
```

Expected: all tests PASS (no fetchEvents/addEvent tests remain).

- [ ] **Step 8: Commit**

```bash
git add \
  api/services/frontend/web/src/stores/contextStore.ts \
  api/services/frontend/web/src/services/contextService.ts \
  api/services/frontend/web/src/composables/useContextDetail.ts \
  api/services/frontend/web/src/__tests__/stores/contextStore.test.ts \
  api/services/frontend/web/src/__tests__/services/contextService.test.ts \
  api/services/frontend/web/src/__tests__/composables/useContextDetail.test.ts
git commit -m "refactor: remove context_events plumbing from store, service, and composable"
```

---

### Task 3: Update ContextDetailView — remove Events collapsible, add sidebar Events card

**Files:**
- Modify: `src/views/ContextDetailView.vue`
- Modify: `src/__tests__/views/ContextDetailView.test.ts`

- [ ] **Step 1: Update the script section of `ContextDetailView.vue`**

Replace the `<script setup>` block with:

```ts
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useContextDetail } from '@/composables/useContextDetail'
import { useTagStore } from '@/stores/tagStore'
import { useNoteStore } from '@/stores/noteStore'
import { useCalendarEventStore } from '@/stores/calendarEventStore'
import { storeToRefs } from 'pinia'
import ContextForm from '@/components/contexts/ContextForm.vue'
import NoteList from '@/components/notes/NoteList.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import CalendarEventCard from '@/components/calendar-events/CalendarEventCard.vue'
import CalendarEventForm from '@/components/calendar-events/CalendarEventForm.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import { observationService, type Observation } from '@/services/observationService'
import { contextService } from '@/services/contextService'
import { ContextKind } from '@/types/enums'
import type { UpdateContext, Note, Context, CalendarEvent, Task } from '@/types'
import type { NewCalendarEvent, UpdateCalendarEvent } from '@/types/calendarEvent'

const route = useRoute()
const router = useRouter()
const contextId = route.params.id as string

const observations = ref<Observation[]>([])
const subContexts = ref<Context[]>([])
const showThread = ref(false)
const showNewSubProject = ref(false)
const showAddEvent = ref(false)

type TimelineItem =
  | { type: 'task'; item: Task; sortKey: string }
  | { type: 'event'; item: CalendarEvent; sortKey: string }

onMounted(async () => {
  observationService.queryBySubject('context', contextId).then((obs) => {
    observations.value = obs
  })
})

const {
  context,
  tags,
  linkedTasks,
  loading,
  update,
  remove,
  addTag,
  removeTag,
} = useContextDetail(contextId)

const tagStore = useTagStore()
const noteStore = useNoteStore()
const calendarEventStore = useCalendarEventStore()
const { items: contextNotes, loading: notesLoading } = storeToRefs(noteStore)
const { items: contextCalendarEvents } = storeToRefs(calendarEventStore)

onMounted(async () => {
  noteStore.setFilter({ contextId })
  noteStore.fetchList(true)
  calendarEventStore.setFilter({ contextId })
  calendarEventStore.fetchList(true)
})

// Fetch sub-contexts when context is an area
onMounted(async () => {
  const result = await contextService.list({ filter: { parentContextId: contextId }, page: 1, rows: 100 })
  subContexts.value = result.items
})

// Progress computed values (project hub)
const doneTasks = computed(() => linkedTasks.value.filter((t) => t.status === 'done').length)
const totalTasks = computed(() => linkedTasks.value.length)

// Combined timeline (project hub)
const timeline = computed<TimelineItem[]>(() => {
  const items: TimelineItem[] = []
  for (const t of linkedTasks.value) {
    items.push({ type: 'task', item: t, sortKey: t.scheduledAt ?? t.dueDate ?? t.createdAt })
  }
  for (const e of contextCalendarEvents.value) {
    items.push({ type: 'event', item: e, sortKey: e.startsAt })
  }
  return items.sort((a, b) => a.sortKey.localeCompare(b.sortKey))
})

function isUnschedulable(task: Task): boolean {
  return task.status === 'open' && !task.scheduledAt && !task.dueDate
}

function formatObsData(data: Record<string, unknown> | unknown): string {
  if (data === null || data === undefined) return ''
  if (typeof data === 'object' && !Array.isArray(data)) {
    return Object.entries(data as Record<string, unknown>)
      .map(([k, v]) => `${k}: ${typeof v === 'string' ? `"${v}"` : String(v)}`)
      .join('\n')
  }
  return String(data)
}

async function handleDeleteContextNote(note: Note) {
  await noteStore.remove(note.id)
}

function handleEditContextNote(note: Note) {
  console.log('edit context note', note.id)
}

const editing = ref(false)
const confirmDelete = ref(false)

async function handleUpdate(data: UpdateContext | Record<string, unknown>) {
  await update(data as UpdateContext)
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'contexts' })
}

async function handleAddTag(tagId: string) {
  await addTag(tagId)
}

async function handleCreateTag(name: string) {
  const tag = await tagStore.create({ name })
  if (tag) {
    await addTag(tag.id)
  }
}

function openTask(id: string) {
  router.push({ name: 'task-detail', params: { id } })
}

async function handleDeleteCalendarEvent(id: string) {
  await calendarEventStore.remove(id)
}

async function handleCreateCalendarEvent(data: NewCalendarEvent | UpdateCalendarEvent) {
  await calendarEventStore.create(data as NewCalendarEvent)
  calendarEventStore.fetchList(true)
  showAddEvent.value = false
}

function navigateToSubContext(id: string) {
  router.push({ name: 'context-detail', params: { id } })
}

async function handleCreateSubProject(data: UpdateContext | Record<string, unknown>) {
  const created = await contextService.create(data as import('@/types').NewContext)
  subContexts.value.push(created)
  showNewSubProject.value = false
}
```

- [ ] **Step 2: Replace the template section of `ContextDetailView.vue`**

Replace the entire `<template>` block with:

```html
<template>
  <div>
    <PageHeader
      :title="context?.title ?? 'Loading...'"
      :subtitle="context?.description"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="router.push({ name: 'contexts' })"
        >
          Back
        </button>
        <button
          v-if="context"
          class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="editing = !editing"
        >
          {{ editing ? 'Cancel' : 'Edit' }}
        </button>
        <button
          v-if="context"
          class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="confirmDelete = true"
        >
          Delete
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading && !context" />

    <div
      v-else-if="context"
      class="p-6"
    >
      <!-- Edit Form -->
      <div
        v-if="editing"
        class="max-w-2xl mb-8"
      >
        <ContextForm
          :context="context"
          mode="edit"
          @submit="handleUpdate"
          @cancel="editing = false"
        />
      </div>

      <!-- Context Info -->
      <div
        v-else
        class="grid grid-cols-1 lg:grid-cols-3 gap-6"
      >
        <!-- ===================== PROJECT HUB ===================== -->
        <template v-if="context.kind === ContextKind.Project">
          <!-- Main Content -->
          <div class="lg:col-span-2 space-y-6">
            <!-- Status & Summary -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div class="flex items-center gap-3 mb-3">
                <StatusBadge
                  :status="context.status"
                  type="context"
                />
                <div class="ml-auto flex items-center gap-2">
                  <div class="h-2 w-24 bg-gray-700 rounded-full overflow-hidden">
                    <div
                      class="h-2 bg-green-500 rounded-full transition-all"
                      :style="{ width: totalTasks > 0 ? `${Math.round((doneTasks / totalTasks) * 100)}%` : '0%' }"
                    />
                  </div>
                  <span class="text-xs text-gray-400">
                    {{ doneTasks }} / {{ totalTasks }} tasks done
                  </span>
                </div>
              </div>
              <div
                v-if="context.summary"
                class="text-sm text-gray-400"
              >
                {{ context.summary }}
              </div>
            </div>

            <!-- Combined Timeline -->
            <div>
              <h3 class="text-sm font-semibold text-gray-300 mb-3 uppercase tracking-wider">
                Timeline
              </h3>
              <div
                v-if="timeline.length === 0"
                class="text-xs text-gray-500"
              >
                No tasks or events yet.
              </div>
              <div class="space-y-2">
                <template
                  v-for="item in timeline"
                  :key="item.type + '-' + item.item.id"
                >
                  <div
                    v-if="item.type === 'task'"
                    class="relative"
                  >
                    <TaskCard
                      :task="item.item"
                      @click="openTask"
                    />
                    <span
                      v-if="isUnschedulable(item.item)"
                      class="absolute top-2 right-2 text-xs bg-amber-500/20 text-amber-400 border border-amber-500/30 rounded px-1.5 py-0.5"
                    >
                      ⚠ No due date
                    </span>
                  </div>
                  <CalendarEventCard
                    v-else-if="item.type === 'event'"
                    :event="item.item"
                    @delete="handleDeleteCalendarEvent"
                  />
                </template>
              </div>
            </div>

            <!-- Collapsible: Activity Thread -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
              <button
                class="w-full flex items-center justify-between px-4 py-3 text-sm font-semibold text-gray-300 uppercase tracking-wider hover:bg-gray-800 transition-colors"
                @click="showThread = !showThread"
              >
                <span>{{ showThread ? '▼' : '▶' }} Thread</span>
              </button>
              <div
                v-if="showThread"
                class="px-4 pb-4"
              >
                <ThreadPanel
                  subject-type="context"
                  :subject-id="contextId"
                />
              </div>
            </div>
          </div>

          <!-- Sidebar (Project) -->
          <div class="space-y-6">
            <!-- Tags -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <h4 class="text-sm font-medium text-gray-300 mb-2">
                Tags
              </h4>
              <TagList
                :tags="tags"
                removable
                @remove="removeTag"
              />
              <TagPicker
                :selected-ids="(tags || []).map(t => t.id)"
                class="mt-2"
                @add="handleAddTag"
                @create="handleCreateTag"
              />
            </div>

            <!-- Events -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div class="flex items-center justify-between mb-2">
                <h4 class="text-sm font-medium text-gray-300">
                  Events
                </h4>
                <button
                  class="px-2 py-0.5 text-xs text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded transition-colors"
                  @click="showAddEvent = true"
                >
                  + Add
                </button>
              </div>
              <div
                v-if="contextCalendarEvents.length === 0"
                class="text-xs text-gray-500"
              >
                No events yet.
              </div>
              <div class="space-y-2">
                <CalendarEventCard
                  v-for="event in contextCalendarEvents"
                  :key="event.id"
                  :event="event"
                  @delete="handleDeleteCalendarEvent"
                />
              </div>
            </div>

            <!-- Observations -->
            <div
              v-if="observations.length > 0"
              class="bg-gray-900 border border-gray-800 rounded-lg p-4"
            >
              <h4 class="text-sm font-medium text-gray-300 mb-3">
                Observations
              </h4>
              <div class="space-y-2">
                <div
                  v-for="obs in observations"
                  :key="obs.id"
                  class="bg-gray-800 rounded-lg p-3"
                >
                  <pre class="text-sm text-gray-200 whitespace-pre-wrap font-sans">{{ formatObsData(obs.data) }}</pre>
                  <p class="text-xs text-gray-500 mt-1">
                    {{ obs.kind }} · {{ obs.source }}
                  </p>
                </div>
              </div>
            </div>

            <!-- Notes -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <h4 class="text-sm font-medium text-gray-300 mb-2">
                Notes
              </h4>
              <NoteList
                :notes="contextNotes"
                :loading="notesLoading"
                @delete="handleDeleteContextNote"
                @edit="handleEditContextNote"
              />
            </div>
          </div>
        </template>

        <!-- ===================== AREA HUB ===================== -->
        <template v-else-if="context.kind === ContextKind.Area">
          <!-- Main Content -->
          <div class="lg:col-span-2 space-y-6">
            <!-- Status & Summary -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div class="flex items-center gap-3 mb-3">
                <StatusBadge
                  :status="context.status"
                  type="context"
                />
              </div>
              <div
                v-if="context.summary"
                class="text-sm text-gray-400"
              >
                {{ context.summary }}
              </div>
            </div>

            <!-- Sub-projects -->
            <div>
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Sub-projects
                </h3>
                <button
                  class="px-2.5 py-1 text-xs text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded-lg transition-colors"
                  @click="showNewSubProject = !showNewSubProject"
                >
                  {{ showNewSubProject ? 'Cancel' : '+ New' }}
                </button>
              </div>
              <div
                v-if="showNewSubProject"
                class="mb-4 bg-gray-900 border border-gray-800 rounded-lg p-4"
              >
                <ContextForm
                  mode="create"
                  :parent-context-id="contextId"
                  @submit="handleCreateSubProject"
                  @cancel="showNewSubProject = false"
                />
              </div>
              <div
                v-if="subContexts.length === 0 && !showNewSubProject"
                class="text-xs text-gray-500"
              >
                No sub-projects yet.
              </div>
              <div class="space-y-2">
                <button
                  v-for="subCtx in subContexts"
                  :key="subCtx.id"
                  class="w-full flex items-center justify-between bg-gray-900 border border-gray-800 rounded-lg px-4 py-3 hover:bg-gray-800 transition-colors text-left"
                  @click="navigateToSubContext(subCtx.id)"
                >
                  <span class="text-sm font-medium text-gray-200">{{ subCtx.title }}</span>
                  <div class="flex items-center gap-2">
                    <StatusBadge
                      :status="subCtx.status"
                      type="context"
                    />
                    <span class="text-gray-500 text-sm">→</span>
                  </div>
                </button>
              </div>
            </div>

            <!-- Floating Tasks -->
            <div>
              <h3 class="text-sm font-semibold text-gray-300 mb-3 uppercase tracking-wider">
                Floating Tasks ({{ linkedTasks.length }})
              </h3>
              <div
                v-if="linkedTasks.length === 0"
                class="text-xs text-gray-500"
              >
                No tasks directly in this area.
              </div>
              <div class="space-y-2">
                <TaskCard
                  v-for="task in linkedTasks"
                  :key="task.id"
                  :task="task"
                  @click="openTask"
                />
              </div>
            </div>

            <!-- Collapsible: Activity Thread -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden">
              <button
                class="w-full flex items-center justify-between px-4 py-3 text-sm font-semibold text-gray-300 uppercase tracking-wider hover:bg-gray-800 transition-colors"
                @click="showThread = !showThread"
              >
                <span>{{ showThread ? '▼' : '▶' }} Thread</span>
              </button>
              <div
                v-if="showThread"
                class="px-4 pb-4"
              >
                <ThreadPanel
                  subject-type="context"
                  :subject-id="contextId"
                />
              </div>
            </div>

            <!-- Collapsible: Observations -->
            <div
              v-if="observations.length > 0"
              class="bg-gray-900 border border-gray-800 rounded-lg overflow-hidden"
            >
              <h4 class="px-4 py-3 text-sm font-medium text-gray-300">
                Observations
              </h4>
              <div class="px-4 pb-4 space-y-2">
                <div
                  v-for="obs in observations"
                  :key="obs.id"
                  class="bg-gray-800 rounded-lg p-3"
                >
                  <pre class="text-sm text-gray-200 whitespace-pre-wrap font-sans">{{ formatObsData(obs.data) }}</pre>
                  <p class="text-xs text-gray-500 mt-1">
                    {{ obs.kind }} · {{ obs.source }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- Sidebar (Area) -->
          <div class="space-y-6">
            <!-- Tags -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <h4 class="text-sm font-medium text-gray-300 mb-2">
                Tags
              </h4>
              <TagList
                :tags="tags"
                removable
                @remove="removeTag"
              />
              <TagPicker
                :selected-ids="(tags || []).map(t => t.id)"
                class="mt-2"
                @add="handleAddTag"
                @create="handleCreateTag"
              />
            </div>

            <!-- Events -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <div class="flex items-center justify-between mb-2">
                <h4 class="text-sm font-medium text-gray-300">
                  Events
                </h4>
                <button
                  class="px-2 py-0.5 text-xs text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded transition-colors"
                  @click="showAddEvent = true"
                >
                  + Add
                </button>
              </div>
              <div
                v-if="contextCalendarEvents.length === 0"
                class="text-xs text-gray-500"
              >
                No events yet.
              </div>
              <div class="space-y-2">
                <CalendarEventCard
                  v-for="event in contextCalendarEvents"
                  :key="event.id"
                  :event="event"
                  @delete="handleDeleteCalendarEvent"
                />
              </div>
            </div>

            <!-- Notes -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
              <h4 class="text-sm font-medium text-gray-300 mb-2">
                Notes
              </h4>
              <NoteList
                :notes="contextNotes"
                :loading="notesLoading"
                @delete="handleDeleteContextNote"
                @edit="handleEditContextNote"
              />
            </div>
          </div>
        </template>
      </div>
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete Context"
      message="Are you sure you want to delete this context? All linked events will also be deleted."
      @confirm="handleDelete"
      @cancel="confirmDelete = false"
    />

    <DrawerPanel
      :open="showAddEvent"
      title="New Event"
      @close="showAddEvent = false"
    >
      <CalendarEventForm
        :initial-context-id="contextId"
        @save="handleCreateCalendarEvent"
        @cancel="showAddEvent = false"
      />
    </DrawerPanel>
  </div>
</template>
```

- [ ] **Step 3: Update `ContextDetailView.test.ts` — remove events/addEvent from mock shape**

In `src/__tests__/views/ContextDetailView.test.ts`, update every `useContextDetail` mock return value. Remove `events`, `eventsTotal`, and `addEvent` from each. The new mock shape is:

```ts
{
  context: computed(() => makeContext({ ... })),
  tags: computed(() => []),
  linkedTasks: computed(() => []),
  loading: computed(() => false),
  update: vi.fn().mockResolvedValue(undefined),
  remove: vi.fn().mockResolvedValue(undefined),
  addTag: vi.fn().mockResolvedValue(undefined),
  removeTag: vi.fn().mockResolvedValue(undefined),
}
```

Apply this shape to all 3 places in the file: the top-level `vi.mock` block and each `vi.mocked(useContextDetail).mockReturnValueOnce(...)` call.

Also add `create: vi.fn().mockResolvedValue({ id: 'evt-1' })` and `fetchList: vi.fn().mockResolvedValue(undefined)` to the `useCalendarEventStore` mock:

```ts
vi.mock('@/stores/calendarEventStore', async () => {
  const { ref } = await import('vue')
  return {
    useCalendarEventStore: () => ({
      items: ref([]),
      loading: ref(false),
      setFilter: vi.fn(),
      fetchList: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      create: vi.fn().mockResolvedValue({ id: 'evt-1' }),
    }),
  }
})
```

- [ ] **Step 4: Run ContextDetailView tests**

```bash
cd api/services/frontend/web && npx vitest run src/__tests__/views/ContextDetailView.test.ts
```

Expected: all tests PASS.

- [ ] **Step 5: Commit**

```bash
git add api/services/frontend/web/src/views/ContextDetailView.vue \
        api/services/frontend/web/src/__tests__/views/ContextDetailView.test.ts
git commit -m "feat: replace context_events section with calendar events sidebar card in context hub"
```

---

### Task 4: Delete legacy event component files

**Files to delete:**
- `src/components/events/EventForm.vue`
- `src/components/events/EventTimeline.vue`
- `src/components/events/EventTimelineItem.vue`
- `src/__tests__/components/events/EventForm.test.ts`
- `src/__tests__/components/events/EventTimeline.test.ts`

- [ ] **Step 1: Delete the files**

```bash
rm api/services/frontend/web/src/components/events/EventForm.vue
rm api/services/frontend/web/src/components/events/EventTimeline.vue
rm api/services/frontend/web/src/components/events/EventTimelineItem.vue
rm api/services/frontend/web/src/__tests__/components/events/EventForm.test.ts
rm api/services/frontend/web/src/__tests__/components/events/EventTimeline.test.ts
```

- [ ] **Step 2: Verify no remaining imports**

```bash
grep -r "EventForm\|EventTimeline\|EventTimelineItem" api/services/frontend/web/src/ --include="*.ts" --include="*.vue"
```

Expected: no output (zero matches).

- [ ] **Step 3: Commit**

```bash
git add -A api/services/frontend/web/src/components/events/
git add -A api/services/frontend/web/src/__tests__/components/events/
git commit -m "chore: delete legacy EventForm, EventTimeline, EventTimelineItem components"
```

---

### Task 5: Remove `addEvent: vi.fn()` from contextStore mocks in remaining test files

These 14 test files mock `useContextStore` or `contextService` with `addEvent`. Remove `addEvent: vi.fn()` from each mock shape.

**Files:**

1. `src/__tests__/components/tasks/TaskCard.test.ts`
2. `src/__tests__/composables/useCapture.test.ts`
3. `src/__tests__/composables/useContextBoard.test.ts`
4. `src/__tests__/stores/captureStore.test.ts`
5. `src/__tests__/views/CaptureView.test.ts`
6. `src/__tests__/views/ContextBoardView.test.ts`
7. `src/__tests__/views/DashboardView.test.ts`
8. `src/__tests__/views/SearchView.test.ts`
9. `src/__tests__/views/TaskBoardView.test.ts`
10. `src/__tests__/views/TodayView.test.ts`
11. `src/__tests__/composables/useDashboard.test.ts`
12. `src/__tests__/composables/useSearch.test.ts`
13. `src/__tests__/composables/useToday.test.ts`

- [ ] **Step 1: Remove `addEvent: vi.fn()` from each file**

In each file listed above, find the `contextStore` mock object (inside `vi.mock(...)`) and remove the `addEvent: vi.fn(),` line. Do not change anything else.

- [ ] **Step 2: Run the full test suite**

```bash
cd api/services/frontend/web && npx vitest run
```

Expected: all tests PASS. If any test fails due to `addEvent` still being called, find and remove that call.

- [ ] **Step 3: Commit**

```bash
git add api/services/frontend/web/src/__tests__/
git commit -m "chore: remove addEvent from contextStore mock shape in all test files"
```

---

### Task 6: Build verification

- [ ] **Step 1: Run frontend lint**

```bash
cd api/services/frontend/web && npx vue-tsc --noEmit
```

Expected: no type errors.

- [ ] **Step 2: Run full test suite**

```bash
cd api/services/frontend/web && npx vitest run
```

Expected: all tests PASS.

- [ ] **Step 3: Run frontend build**

```bash
make frontend-build
```

Expected: build succeeds with no errors.

- [ ] **Step 4: Close the beads issue and push**

```bash
bd update planner-azi --claim
bd close planner-azi
git pull --rebase && bd dolt push && git push
```
