<script setup lang="ts">
import { ref, computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTaskDetail } from '@/composables/useTaskDetail'
import { useRelatedByContext } from '@/composables/useRelatedByContext'
import { useTagStore } from '@/stores/tagStore'
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import { useTaskStore } from '@/stores/taskStore'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import ActivityLogButton from '@/components/shared/ActivityLogButton.vue'
import StreakDisplay from '@/components/shared/StreakDisplay.vue'
import ActivityHistory from '@/components/shared/ActivityHistory.vue'
import { correctionService } from '@/services/correctionService'
import { taskService } from '@/services/taskService'
import type { UpdateTask, EntityLink } from '@/types'

const route = useRoute()
const router = useRouter()
const taskId = route.params.id as string

const { task, tags, loading, update, remove, addTag, removeTag } = useTaskDetail(taskId)
const tagStore = useTagStore()
const entityLinkStore = useEntityLinkStore()
const taskStore = useTaskStore()

const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => task.value?.contextId),
  'task',
  taskId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)

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

const editing = ref(false)
const confirmDelete = ref(false)
const showConvertConfirm = ref(false)
const correcting = ref(false)
const converting = ref(false)

async function handleDemote(newType: 'note' | 'event') {
  if (!task.value?.id || correcting.value) return
  correcting.value = true
  try {
    await correctionService.correct(task.value.id, 'task', newType)
    router.push({ name: newType === 'note' ? 'notes' : 'tasks' })
  } finally {
    correcting.value = false
  }
}

async function handleUpdate(data: UpdateTask | Record<string, unknown>) {
  await update(data as UpdateTask)
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'tasks' })
}

async function handleConvertToNote() {
  if (!task.value?.id || converting.value) return
  converting.value = true
  try {
    const newNote = await taskService.convertTaskToNote(task.value.id)
    taskStore.remove(task.value.id)
    showConvertConfirm.value = false
    const navTo: Record<string, unknown> = {
      name: 'note-detail',
      params: { id: newNote.id },
    }
    if (route.query.context) {
      navTo.query = { context: route.query.context }
    }
    router.push(navTo)
  } finally {
    converting.value = false
  }
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
              class="px-3 py-1.5 text-sm text-purple-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="showConvertConfirm = true"
            >
              Convert to Note
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

        <div class="space-y-3 text-sm">
          <div class="flex justify-between">
            <span class="text-gray-500">Status</span>
            <span class="text-gray-300">{{ task.status }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Priority</span>
            <span class="text-gray-300">{{ task.priority }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Energy</span>
            <span class="text-gray-300">{{ task.energy }}</span>
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
            <router-link
              :to="{ name: 'task-detail', params: { id: task.recurrenceParentId } }"
              class="text-blue-400 hover:text-blue-300"
            >
              View parent
            </router-link>
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
            @remove="removeTag"
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

    <ConfirmDialog
      :open="showConvertConfirm"
      title="Convert to Note"
      message="Convert this task to a note? Tags and context will be preserved."
      :loading="converting"
      @confirm="handleConvertToNote"
      @cancel="showConvertConfirm = false"
    />
  </div>
</template>
