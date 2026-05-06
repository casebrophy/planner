<script setup lang="ts">
import { ref, computed, watch, watchEffect } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useTagStore } from '@/stores/tagStore'
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import { useRelatedByContext } from '@/composables/useRelatedByContext'
import { useTaskDrawer } from '@/composables/useTaskDrawer'
import { observationService } from '@/services/observationService'
import { correctionService } from '@/services/correctionService'
import {
  TaskStatus, TaskStatusLabels, StatusColors,
  TaskPriority, TaskPriorityLabels,
} from '@/types/enums'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TaskDebriefDialog from '@/components/tasks/TaskDebriefDialog.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import ActivityLogButton from '@/components/shared/ActivityLogButton.vue'
import StreakDisplay from '@/components/shared/StreakDisplay.vue'
import ActivityHistory from '@/components/shared/ActivityHistory.vue'
import type { UpdateTask, EntityLink, Task } from '@/types'

const props = defineProps<{
  taskId: string
}>()

const emit = defineEmits<{
  close: []
}>()

const taskStore = useTaskStore()
const tagStore = useTagStore()
const entityLinkStore = useEntityLinkStore()
const { open: openDrawer } = useTaskDrawer()

const loading = ref(true)
const task = computed(() => taskStore.currentItem)
const tags = computed(() => tagStore.taskTags[props.taskId] ?? [])

const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => task.value?.contextId),
  'task',
  props.taskId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)

// Load task data on mount
watch(() => props.taskId, async (id) => {
  if (!id) return
  loading.value = true
  await Promise.all([
    taskStore.fetchById(id),
    tagStore.fetchTagsForTask(id),
  ])
  loading.value = false
}, { immediate: true })

watchEffect(async () => {
  if (task.value?.id) {
    await entityLinkStore.fetchLinks('task', task.value.id)
  }
})

const explicitLinks = computed(() => {
  if (!task.value?.id) return []
  return entityLinkStore.getLinks('task', task.value.id)
})

const showLinkModal = ref(false)
const linkTargetType = ref<'task' | 'note' | 'event'>('task')
const linkTargetId = ref('')
const editing = ref(false)
const showDebrief = ref(false)
const confirmDelete = ref(false)
const correcting = ref(false)

// Inline field update
async function updateField(field: string, value: string | number | undefined) {
  if (!task.value) return
  const data: UpdateTask = { [field]: value }
  await taskStore.update(props.taskId, data)
  if (field === 'status' && value === 'done') {
    const skip = await shouldAutoSkip(task.value)
    if (!skip) {
      showDebrief.value = true
    }
  }
}

const statusOptions = [TaskStatus.Open, TaskStatus.Blocked, TaskStatus.Done, TaskStatus.Dismissed]
const priorityOptions = [TaskPriority.Low, TaskPriority.Medium, TaskPriority.High, TaskPriority.Urgent]

const durationInput = ref('')
watch(() => task.value?.durationMin, (val) => {
  durationInput.value = val ? String(val) : ''
}, { immediate: true })

function commitDuration() {
  const parsed = parseInt(durationInput.value, 10)
  const newVal = isNaN(parsed) || parsed <= 0 ? undefined : parsed
  if (newVal !== task.value?.durationMin) {
    updateField('durationMin', newVal)
  }
}

async function addLink() {
  if (!task.value?.id || !linkTargetId.value.trim()) return
  await entityLinkStore.createLink({
    sourceType: 'task',
    sourceId: task.value.id,
    targetType: linkTargetType.value,
    targetId: linkTargetId.value.trim(),
  })
  showLinkModal.value = false
  linkTargetId.value = ''
}

async function removeLink(link: EntityLink) {
  await entityLinkStore.deleteLink(link)
}

async function handleDemote(newType: 'note' | 'event') {
  if (!task.value?.id || correcting.value) return
  correcting.value = true
  try {
    await correctionService.correct(task.value.id, 'task', newType)
    emit('close')
  } finally {
    correcting.value = false
  }
}

async function shouldAutoSkip(currentTask: Task | undefined): Promise<boolean> {
  if (!currentTask) return false
  const observations = await observationService.queryByKind('task', 'debrief')
  const matching = observations.filter(o => {
    const d = o.data as { contextId?: string; priority?: string }
    return d.contextId === (currentTask.contextId || null)
      && d.priority === currentTask.priority
  })
  if (matching.length < 5) return false
  const skipped = matching.filter(o => (o.data as { outcome: string }).outcome === 'skipped')
  return skipped.length / matching.length > 0.8
}

async function handleUpdate(data: UpdateTask | Record<string, unknown>) {
  await taskStore.update(props.taskId, data as UpdateTask)
  if ((data as UpdateTask).status === 'done') {
    const skip = await shouldAutoSkip(task.value || undefined)
    if (!skip) {
      showDebrief.value = true
    }
  }
  editing.value = false
}

async function handleDelete() {
  await taskStore.remove(props.taskId)
  confirmDelete.value = false
  emit('close')
}

async function handleAddTag(tagId: string) {
  await tagStore.addTagToTask(props.taskId, tagId)
}

async function handleCreateTag(name: string) {
  const tag = await tagStore.create({ name })
  if (tag) {
    await tagStore.addTagToTask(props.taskId, tag.id)
  }
}

async function handleRemoveTag(tagId: string) {
  await tagStore.removeTagFromTask(props.taskId, tagId)
}

function openRelatedTask(id: string) {
  openDrawer(id)
}
</script>

<template>
  <div>
    <LoadingSpinner
      v-if="loading && !task"
      size="sm"
    />

    <div
      v-else-if="task"
      class="space-y-6"
    >
      <!-- View Mode -->
      <div v-if="!editing">
        <!-- Unconfirmed classification banner -->
        <div
          v-if="task.unconfirmed"
          class="mb-4 p-3 rounded-lg bg-amber-900/20 border border-amber-700/40"
        >
          <p class="text-sm text-amber-300 mb-2">
            This item was created with low classifier confidence. Is it really a task?
          </p>
          <div class="flex gap-2">
            <button
              :disabled="correcting"
              class="px-3 py-1.5 text-xs font-medium text-white bg-violet-600 hover:bg-violet-500 rounded-lg transition-colors disabled:opacity-40"
              @click="handleDemote('note')"
            >
              Move to Note
            </button>
            <button
              :disabled="correcting"
              class="px-3 py-1.5 text-xs font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40"
              @click="handleDemote('event')"
            >
              Move to Event
            </button>
          </div>
        </div>

        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-gray-100">
            {{ task.title }}
          </h3>
          <div class="flex gap-2">
            <button
              class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="editing = true"
            >
              Edit
            </button>
            <button
              class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="confirmDelete = true"
            >
              Delete
            </button>
          </div>
        </div>

        <p
          v-if="task.description"
          class="text-sm text-gray-400 mb-4"
        >
          {{ task.description }}
        </p>

        <!-- Inline-editable fields -->
        <div class="space-y-3 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-gray-500">Status</span>
            <select
              :value="task.status"
              class="bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-2 py-1 text-sm focus:outline-none focus:border-indigo-500 cursor-pointer"
              @change="updateField('status', ($event.target as HTMLSelectElement).value)"
            >
              <option
                v-for="opt in statusOptions"
                :key="opt"
                :value="opt"
                :style="{ color: StatusColors[opt] }"
              >
                {{ TaskStatusLabels[opt] }}
              </option>
            </select>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500">Priority</span>
            <select
              :value="task.priority"
              class="bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-2 py-1 text-sm focus:outline-none focus:border-indigo-500 cursor-pointer"
              @change="updateField('priority', ($event.target as HTMLSelectElement).value)"
            >
              <option
                v-for="opt in priorityOptions"
                :key="opt"
                :value="opt"
              >
                {{ TaskPriorityLabels[opt] }}
              </option>
            </select>
          </div>
          <div class="flex items-center justify-between">
            <span class="text-gray-500">Time Estimate</span>
            <div class="flex items-center gap-1">
              <input
                v-model="durationInput"
                type="number"
                min="0"
                placeholder="—"
                class="w-16 bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-2 py-1 text-sm text-right focus:outline-none focus:border-indigo-500"
                @blur="commitDuration"
                @keyup.enter="($event.target as HTMLInputElement).blur()"
              >
              <span class="text-gray-500 text-xs">min</span>
            </div>
          </div>
          <div
            v-if="task.dueDate"
            class="flex justify-between"
          >
            <span class="text-gray-500">Due</span>
            <span class="text-gray-300">{{ new Date(task.dueDate).toLocaleDateString() }}</span>
          </div>
          <div
            v-if="task.recurrenceRule"
            class="flex justify-between"
          >
            <span class="text-gray-500">Recurrence</span>
            <span class="text-gray-300">{{ task.recurrenceRule }}</span>
          </div>
          <div
            v-if="task.recurrenceParentId"
            class="flex justify-between"
          >
            <span class="text-gray-500">Parent Task</span>
            <button
              class="text-blue-400 hover:text-blue-300"
              @click="openRelatedTask(task.recurrenceParentId!)"
            >
              View parent
            </button>
          </div>
        </div>

        <!-- Tags -->
        <div class="mt-6">
          <h4 class="text-sm font-medium text-gray-300 mb-2">
            Tags
          </h4>
          <TagList
            :tags="tags"
            removable
            @remove="handleRemoveTag"
          />
          <TagPicker
            :selected-ids="tags.map(t => t.id)"
            class="mt-2"
            @add="handleAddTag"
            @create="handleCreateTag"
          />
        </div>

        <!-- Activity Tracking -->
        <div class="mt-6">
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-sm font-medium text-gray-300">
              Activity
            </h4>
            <ActivityLogButton
              subject-type="task"
              :subject-id="taskId"
            />
          </div>
          <StreakDisplay
            subject-type="task"
            :subject-id="taskId"
          />
          <div class="mt-3">
            <ActivityHistory
              subject-type="task"
              :subject-id="taskId"
            />
          </div>
        </div>

        <!-- Related Items Panel -->
        <div class="mt-6">
          <h3 class="text-sm font-semibold text-gray-400 uppercase tracking-wide mb-3">
            Related Items
          </h3>

          <div
            v-if="hasSameContextItems"
            class="mb-4"
          >
            <h4 class="text-xs font-medium text-gray-500 mb-2">
              In same context
            </h4>
            <div class="flex flex-col gap-2">
              <button
                v-for="t in sameContextTasks"
                :key="t.id"
                class="flex items-center bg-gray-700 rounded-lg px-3 py-2 text-sm text-gray-300 hover:bg-gray-600 transition-colors text-left"
                @click="openRelatedTask(t.id)"
              >
                <span class="text-gray-500 mr-2 text-xs">task</span>
                {{ t.title }}
              </button>
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

          <div
            v-if="explicitLinks.length > 0"
            class="flex flex-col gap-2 mb-4"
          >
            <h4 class="text-xs font-medium text-gray-500 mb-2">
              Also related
            </h4>
            <div
              v-for="link in explicitLinks"
              :key="link.id"
              class="flex items-center justify-between bg-gray-700 rounded-lg px-3 py-2 text-sm"
            >
              <span class="text-gray-300">
                {{ link.sourceId === task.id ? link.targetType : link.sourceType }}:
                {{ link.sourceId === task.id ? link.targetId : link.sourceId }}
              </span>
              <button
                class="text-gray-500 hover:text-red-400 transition-colors"
                aria-label="Remove link"
                @click="removeLink(link)"
              >
                ×
              </button>
            </div>
          </div>

          <div v-if="!showLinkModal">
            <button
              class="px-3 py-1.5 text-xs text-gray-400 border border-gray-600 hover:border-gray-500 rounded-lg transition-colors"
              @click="showLinkModal = true"
            >
              + Link manually
            </button>
          </div>

          <div
            v-else
            class="flex flex-col gap-2"
          >
            <div class="flex gap-2">
              <select
                v-model="linkTargetType"
                class="bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-2 py-1.5 text-sm focus:outline-none focus:border-indigo-500"
              >
                <option value="task">
                  Task
                </option>
                <option value="note">
                  Note
                </option>
                <option value="event">
                  Event
                </option>
              </select>
              <input
                v-model="linkTargetId"
                placeholder="Target ID"
                class="flex-1 bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-indigo-500"
              >
            </div>
            <div class="flex gap-2">
              <button
                class="px-3 py-1.5 text-xs text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors disabled:opacity-40"
                :disabled="!linkTargetId.trim()"
                @click="addLink"
              >
                Link
              </button>
              <button
                class="px-3 py-1.5 text-xs text-gray-400 border border-gray-600 hover:border-gray-500 rounded-lg transition-colors"
                @click="showLinkModal = false; linkTargetId = ''"
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Edit Mode -->
      <TaskForm
        v-else
        :task="task"
        mode="edit"
        @submit="handleUpdate"
        @cancel="editing = false"
      />
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete Task"
      message="Are you sure you want to delete this task? This action cannot be undone."
      @confirm="handleDelete"
      @cancel="confirmDelete = false"
    />

    <TaskDebriefDialog
      :open="showDebrief"
      :task-id="taskId"
      :task="task!"
      @close="showDebrief = false"
    />
  </div>
</template>
