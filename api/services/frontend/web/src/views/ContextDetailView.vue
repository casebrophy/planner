<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useContextDetail } from '@/composables/useContextDetail'
import { useTaskDrawer } from '@/composables/useTaskDrawer'
import { useTagStore } from '@/stores/tagStore'
import { useNoteStore } from '@/stores/noteStore'
import { useCalendarEventStore } from '@/stores/calendarEventStore'
import { storeToRefs } from 'pinia'
import ContextForm from '@/components/contexts/ContextForm.vue'
import NoteList from '@/components/notes/NoteList.vue'
import CalendarEventForm from '@/components/calendar-events/CalendarEventForm.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import CalendarEventCard from '@/components/calendar-events/CalendarEventCard.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import { observationService, type Observation } from '@/services/observationService'
import { contextService } from '@/services/contextService'
import { taskService } from '@/services/taskService'
import { ContextKind } from '@/types/enums'
import type { UpdateContext, Note, Context, CalendarEvent, Task, NewTask, UpdateTask } from '@/types'

const route = useRoute()
const router = useRouter()
const contextId = route.params.id as string

const observations = ref<Observation[]>([])
const subContexts = ref<Context[]>([])
const showAddEvent = ref(false)
const showThread = ref(false)
const showNewSubProject = ref(false)
const showNewTask = ref(false)

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
  reload,
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

onUnmounted(() => {
  noteStore.setFilter({})
  calendarEventStore.setFilter({})
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

const editingCalendarEvent = ref<CalendarEvent | null>(null)

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

function openEditCalendarEvent(event: CalendarEvent) {
  editingCalendarEvent.value = event
  showAddEvent.value = true
}

async function handleSaveCalendarEvent(data: import('@/types').NewCalendarEvent | import('@/types').UpdateCalendarEvent) {
  if (editingCalendarEvent.value) {
    await calendarEventStore.update(editingCalendarEvent.value.id, data as import('@/types').UpdateCalendarEvent)
  } else {
    await calendarEventStore.create(data as import('@/types').NewCalendarEvent)
  }
  await calendarEventStore.fetchList(true)
  showAddEvent.value = false
  editingCalendarEvent.value = null
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

const { open: openTaskDrawer } = useTaskDrawer()

function openTask(id: string) {
  openTaskDrawer(id)
}

async function handleDeleteCalendarEvent(id: string) {
  await calendarEventStore.remove(id)
}

function navigateToSubContext(id: string) {
  router.push({ name: 'context-detail', params: { id } })
}

async function handleCreateSubProject(data: UpdateContext | Record<string, unknown>) {
  const created = await contextService.create(data as import('@/types').NewContext)
  subContexts.value.push(created)
  showNewSubProject.value = false
}

async function handleCreateTask(data: NewTask | UpdateTask) {
  await taskService.create(data as NewTask)
  showNewTask.value = false
  await reload()
}

// List kind state
const newItemTitle = ref('')
const bulkEditMode = ref(false)
const selectedTaskIds = ref<Set<string>>(new Set())
const resetListLoading = ref(false)

const hasDoneTasks = computed(() => linkedTasks.value.some((t) => t.status === 'done'))

async function toggleTaskStatus(task: Task) {
  const newStatus = task.status === 'done' ? 'open' : 'done'
  await taskService.update(task.id, { status: newStatus } as UpdateTask)
  await reload()
}

async function addListItem() {
  const title = newItemTitle.value.trim()
  if (!title) return
  await taskService.create({ title, contextId } as NewTask)
  newItemTitle.value = ''
  await reload()
}

async function resetList() {
  resetListLoading.value = true
  try {
    await contextService.resetList(contextId)
    await reload()
  } finally {
    resetListLoading.value = false
  }
}

function toggleBulkSelect(taskId: string) {
  const next = new Set(selectedTaskIds.value)
  if (next.has(taskId)) {
    next.delete(taskId)
  } else {
    next.add(taskId)
  }
  selectedTaskIds.value = next
}

function selectAll() {
  selectedTaskIds.value = new Set(linkedTasks.value.map((t) => t.id))
}

function selectDone() {
  selectedTaskIds.value = new Set(linkedTasks.value.filter((t) => t.status === 'done').map((t) => t.id))
}

function selectOpen() {
  selectedTaskIds.value = new Set(linkedTasks.value.filter((t) => t.status === 'open').map((t) => t.id))
}

async function deleteSelected() {
  const ids = Array.from(selectedTaskIds.value)
  if (ids.length === 0) return
  await taskService.deleteBatch(ids)
  selectedTaskIds.value = new Set()
  bulkEditMode.value = false
  await reload()
}

function toggleBulkEdit() {
  if (bulkEditMode.value) {
    bulkEditMode.value = false
    selectedTaskIds.value = new Set()
  } else {
    bulkEditMode.value = true
  }
}
</script>

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
                <!-- Progress indicator -->
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
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Timeline
                </h3>
                <button
                  class="px-2.5 py-1 text-xs text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded-lg transition-colors"
                  @click="showNewTask = true"
                >
                  + New Task
                </button>
              </div>
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
                  <!-- Task item -->
                  <TaskCard
                    v-if="item.type === 'task'"
                    :task="item.item"
                    @click="openTask"
                  />
                  <!-- Calendar event item -->
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
                No events yet
              </div>
              <div class="space-y-2">
                <CalendarEventCard
                  v-for="evt in contextCalendarEvents"
                  :key="evt.id"
                  :event="evt"
                  @click="openEditCalendarEvent"
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

            <!-- Sub-projects: only shown for top-level areas (no parent) -->
            <div v-if="!context.parentContextId">
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
              <div class="flex items-center justify-between mb-3">
                <h3 class="text-sm font-semibold text-gray-300 uppercase tracking-wider">
                  Floating Tasks ({{ linkedTasks.length }})
                </h3>
                <button
                  class="px-2.5 py-1 text-xs text-blue-400 bg-blue-500/10 hover:bg-blue-500/20 border border-blue-500/30 rounded-lg transition-colors"
                  @click="showNewTask = true"
                >
                  + New Task
                </button>
              </div>
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
                No events yet
              </div>
              <div class="space-y-2">
                <CalendarEventCard
                  v-for="evt in contextCalendarEvents"
                  :key="evt.id"
                  :event="evt"
                  @click="openEditCalendarEvent"
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

        <!-- ===================== LIST HUB (Checklist) ===================== -->
        <template v-else-if="context.kind === ContextKind.List">
          <div class="lg:col-span-3 space-y-4">
            <!-- Header: status + actions -->
            <div class="flex items-center gap-3">
              <StatusBadge
                :status="context.status"
                type="context"
              />
              <button
                v-if="hasDoneTasks"
                :disabled="resetListLoading"
                class="px-3 py-1.5 text-xs text-teal-400 bg-teal-500/10 hover:bg-teal-500/20 border border-teal-500/30 rounded-lg transition-colors disabled:opacity-50"
                @click="resetList"
              >
                {{ resetListLoading ? 'Resetting...' : 'Reset List' }}
              </button>
              <button
                class="ml-auto px-3 py-1.5 text-xs text-gray-400 bg-gray-800 hover:bg-gray-700 border border-gray-700 rounded-lg transition-colors"
                @click="toggleBulkEdit"
              >
                {{ bulkEditMode ? 'Done' : 'Select' }}
              </button>
            </div>

            <!-- Bulk edit toolbar -->
            <div
              v-if="bulkEditMode"
              class="flex items-center gap-2 bg-gray-900 border border-gray-800 rounded-lg px-4 py-2"
            >
              <button
                class="px-2 py-1 text-xs text-gray-400 hover:text-gray-200 transition-colors"
                @click="selectAll"
              >
                All
              </button>
              <button
                class="px-2 py-1 text-xs text-gray-400 hover:text-gray-200 transition-colors"
                @click="selectDone"
              >
                Done
              </button>
              <button
                class="px-2 py-1 text-xs text-gray-400 hover:text-gray-200 transition-colors"
                @click="selectOpen"
              >
                Open
              </button>
              <span class="text-gray-600 text-xs ml-1">{{ selectedTaskIds.size }} selected</span>
              <button
                v-if="selectedTaskIds.size > 0"
                class="ml-auto px-3 py-1 text-xs text-red-400 bg-red-500/10 hover:bg-red-500/20 border border-red-500/30 rounded-lg transition-colors"
                @click="deleteSelected"
              >
                Delete selected
              </button>
            </div>

            <!-- Checklist items -->
            <div class="bg-gray-900 border border-gray-800 rounded-lg divide-y divide-gray-800">
              <div
                v-if="linkedTasks.length === 0"
                class="px-4 py-6 text-sm text-gray-500 text-center"
              >
                No items yet. Add one below.
              </div>
              <div
                v-for="task in linkedTasks"
                :key="task.id"
                class="flex items-center gap-3 px-4 py-3 hover:bg-gray-800/50 transition-colors"
              >
                <!-- Bulk select checkbox -->
                <input
                  v-if="bulkEditMode"
                  type="checkbox"
                  :checked="selectedTaskIds.has(task.id)"
                  class="w-4 h-4 rounded border-gray-600 bg-gray-700 text-teal-500 focus:ring-teal-500 cursor-pointer"
                  @change="toggleBulkSelect(task.id)"
                />
                <!-- Done/open toggle -->
                <button
                  class="flex-shrink-0 w-5 h-5 rounded border-2 flex items-center justify-center transition-colors"
                  :class="task.status === 'done'
                    ? 'bg-teal-600 border-teal-600 text-white'
                    : 'border-gray-600 hover:border-teal-500'"
                  @click="toggleTaskStatus(task)"
                >
                  <span
                    v-if="task.status === 'done'"
                    class="text-xs leading-none"
                  >✓</span>
                </button>
                <span
                  class="text-sm flex-1"
                  :class="task.status === 'done' ? 'line-through text-gray-500' : 'text-gray-200'"
                >{{ task.title }}</span>
              </div>

              <!-- Inline add-item input -->
              <div class="flex items-center gap-3 px-4 py-3">
                <div class="w-5 h-5 flex-shrink-0" />
                <input
                  v-model="newItemTitle"
                  type="text"
                  placeholder="Add item…"
                  class="flex-1 bg-transparent text-sm text-gray-300 placeholder-gray-600 outline-none"
                  @keydown.enter="addListItem"
                />
                <button
                  v-if="newItemTitle.trim()"
                  class="px-2 py-0.5 text-xs text-teal-400 bg-teal-500/10 hover:bg-teal-500/20 border border-teal-500/30 rounded transition-colors"
                  @click="addListItem"
                >
                  Add
                </button>
              </div>
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
      :open="showNewTask"
      title="New Task"
      @close="showNewTask = false"
    >
      <div class="p-6">
        <TaskForm
          mode="create"
          :initial-context-id="contextId"
          @submit="handleCreateTask"
          @cancel="showNewTask = false"
        />
      </div>
    </DrawerPanel>

    <DrawerPanel
      :open="showAddEvent"
      :title="editingCalendarEvent ? 'Edit Event' : 'Add Event'"
      @close="showAddEvent = false; editingCalendarEvent = null"
    >
      <div class="p-6">
        <CalendarEventForm
          :event="editingCalendarEvent"
          :initial-context-id="editingCalendarEvent ? undefined : contextId"
          @save="handleSaveCalendarEvent"
          @cancel="showAddEvent = false; editingCalendarEvent = null"
        />
      </div>
    </DrawerPanel>
  </div>
</template>
