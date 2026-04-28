<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useTaskBoard } from '@/composables/useTaskBoard'
import { useTaskStore } from '@/stores/taskStore'
import { useContextStore } from '@/stores/contextStore'
import { useTaskDrawer } from '@/composables/useTaskDrawer'
import { ContextKindColors, ContextKindLabels, type ContextKind } from '@/types'
import type { NewTask, UpdateTask } from '@/types'
import PageHeader from '@/components/layout/PageHeader.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskFilterBar from '@/components/tasks/TaskFilterBar.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import Pagination from '@/components/shared/Pagination.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import ClassifyDialog from '@/components/tasks/ClassifyDialog.vue'

const {
  tasks,
  total,
  page,
  loading,
  filter,
  isEmpty,
  pagination,
  setFilter,
  setOrder,
  setPage,
  refresh,
} = useTaskBoard()

const taskStore = useTaskStore()
const contextStore = useContextStore()
const route = useRoute()
const { open: openTaskDrawer } = useTaskDrawer()

const showCreateForm = ref(false)
const showClassify = ref(false)
const groupByContext = ref(true)

// Order by context_id when grouped so a context's tasks stay contiguous across
// pages. Set synchronously during setup so useTaskBoard's onMounted fetch picks
// it up on the initial load.
taskStore.setOrder(groupByContext.value ? 'context_id' : 'created_at')

onMounted(() => {
  if (contextStore.items.length === 0) {
    contextStore.fetchList()
  }
  // Open drawer if navigated to /tasks/:id directly
  const id = route.params.id as string | undefined
  if (id) {
    openTaskDrawer(id)
  }
})

watch(groupByContext, (grouped) => {
  setOrder(grouped ? 'context_id' : 'created_at')
})

async function handleCreate(data: NewTask | UpdateTask) {
  await taskStore.create(data as NewTask)
  showCreateForm.value = false
}

// Groups tasks by contextId. Returns array of { context, tasks } sorted:
// contexts with tasks first (by title), unassigned last.
const groupedTasks = computed(() => {
  const groups = new Map<string | null, typeof tasks.value>()

  for (const task of tasks.value) {
    const key = task.contextId ?? null
    if (!groups.has(key)) groups.set(key, [])
    groups.get(key)!.push(task)
  }

  const result: Array<{ contextId: string | null; contextTitle: string; kind: ContextKind | null; tasks: typeof tasks.value }> = []

  for (const [contextId, taskList] of groups) {
    if (contextId === null) continue
    const ctx = contextStore.contextById(contextId)
    result.push({
      contextId,
      contextTitle: ctx?.title ?? 'Unknown Context',
      kind: ctx?.kind ?? null,
      tasks: taskList,
    })
  }

  // Sort by title
  result.sort((a, b) => a.contextTitle.localeCompare(b.contextTitle))

  // Unassigned last
  const unassigned = groups.get(null)
  if (unassigned?.length) {
    result.push({ contextId: null, contextTitle: 'Unassigned', kind: null, tasks: unassigned })
  }

  return result
})

function openTask(id: string) {
  openTaskDrawer(id)
}
</script>

<template>
  <div>
    <PageHeader
      title="Tasks"
      :subtitle="`${total} tasks`"
    >
      <template #actions>
        <!-- Group toggle -->
        <button
          class="px-3 py-1.5 text-sm font-medium rounded-lg border transition-colors"
          :class="groupByContext
            ? 'bg-blue-600/20 text-blue-400 border-blue-600/40 hover:bg-blue-600/30'
            : 'text-gray-300 bg-gray-800 border-gray-700 hover:bg-gray-700'"
          @click="groupByContext = !groupByContext"
        >
          {{ groupByContext ? 'Grouped' : 'Flat' }}
        </button>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="showClassify = true"
        >
          Classify
        </button>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
        <button
          class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
          @click="showCreateForm = true"
        >
          New Task
        </button>
      </template>
    </PageHeader>

    <div class="p-6">
      <TaskFilterBar
        :filter="filter"
        class="mb-4"
        @update="setFilter"
      />

      <LoadingSpinner v-if="loading && tasks.length === 0" />

      <EmptyState
        v-else-if="isEmpty"
        title="No tasks found"
        message="Create your first task to get started"
        action-label="New Task"
        @action="showCreateForm = true"
      />

      <!-- Grouped view -->
      <div
        v-else-if="groupByContext"
        class="space-y-8"
      >
        <div
          v-for="group in groupedTasks"
          :key="group.contextId ?? 'unassigned'"
        >
          <!-- Group header -->
          <div class="flex items-center gap-2 mb-3">
            <h2 class="text-sm font-semibold text-gray-200">
              {{ group.contextTitle }}
            </h2>
            <span
              v-if="group.kind"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium"
              :style="{
                backgroundColor: (ContextKindColors[group.kind] ?? '#6b7280') + '20',
                color: ContextKindColors[group.kind] ?? '#6b7280'
              }"
            >
              {{ ContextKindLabels[group.kind] ?? group.kind }}
            </span>
            <span class="text-xs text-gray-500">({{ group.tasks.length }})</span>
          </div>
          <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
            <TaskCard
              v-for="task in group.tasks"
              :key="task.id"
              :task="task"
              @click="openTask"
            />
          </div>
        </div>
      </div>

      <!-- Flat view -->
      <div
        v-else
        class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3"
      >
        <TaskCard
          v-for="task in tasks"
          :key="task.id"
          :task="task"
          @click="openTask"
        />
      </div>

      <Pagination
        v-if="pagination.totalPages.value > 1"
        :page="page"
        :total-pages="pagination.totalPages.value"
        :has-next="pagination.hasNextPage.value"
        :has-prev="pagination.hasPrevPage.value"
        class="mt-4"
        @next="setPage(page + 1)"
        @prev="setPage(page - 1)"
      />
    </div>

    <!-- Create Form Drawer -->
    <DrawerPanel
      :open="showCreateForm"
      title="New Task"
      @close="showCreateForm = false"
    >
      <TaskForm
        mode="create"
        @submit="handleCreate"
        @cancel="showCreateForm = false"
      />
    </DrawerPanel>

    <ClassifyDialog
      :open="showClassify"
      @close="showClassify = false; refresh()"
    />
  </div>
</template>
