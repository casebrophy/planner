<script setup lang="ts">
import { ref, computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTaskDetail } from '@/composables/useTaskDetail'
import { useTaskNotes } from '@/composables/useTaskNotes'
import { useRelatedByContext } from '@/composables/useRelatedByContext'
import { useTagStore } from '@/stores/tagStore'
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TaskDebriefDialog from '@/components/tasks/TaskDebriefDialog.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import NoteList from '@/components/notes/NoteList.vue'
import NoteForm from '@/components/notes/NoteForm.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import ActivityLogButton from '@/components/shared/ActivityLogButton.vue'
import StreakDisplay from '@/components/shared/StreakDisplay.vue'
import ActivityHistory from '@/components/shared/ActivityHistory.vue'
import type { UpdateTask, EntityLink, Note, NewNote, UpdateNote } from '@/types'

const route = useRoute()
const router = useRouter()
const taskId = route.params.id as string

const { task, tags, loading, update, remove, addTag, removeTag } = useTaskDetail(taskId)
const { notes, loading: notesLoading, addNote, updateNote, deleteNote } = useTaskNotes(taskId)
const tagStore = useTagStore()
const entityLinkStore = useEntityLinkStore()

const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => task.value?.contextId),
  'task',
  taskId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)

const showNoteForm = ref(false)
const editingNote = ref<Note | null>(null)

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
const showDebrief = ref(false)
const confirmDelete = ref(false)

async function handleUpdate(data: UpdateTask | Record<string, unknown>) {
  await update(data as UpdateTask)
  if ((data as UpdateTask).status === 'done' && task.value?.trackOutcome) {
    showDebrief.value = true
  }
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'tasks' })
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

async function handleDeleteNote(note: Note) {
  await deleteNote(note.id)
}

function handleEditNote(note: Note) {
  editingNote.value = note
  showNoteForm.value = true
}

async function handleNoteSubmit(data: NewNote | UpdateNote) {
  if (editingNote.value) {
    await updateNote(editingNote.value.id, data as UpdateNote)
    editingNote.value = null
  } else {
    await addNote(data as NewNote)
  }
  showNoteForm.value = false
}

function handleNoteCancel() {
  showNoteForm.value = false
  editingNote.value = null
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

        <!-- Notes -->
        <div class="mt-6">
          <div class="flex items-center justify-between mb-2">
            <h4 class="text-sm font-medium text-gray-300">
              Notes
            </h4>
            <button
              v-if="!showNoteForm"
              class="text-xs text-gray-400 border border-gray-600 hover:border-gray-500 px-2 py-1 rounded-lg transition-colors"
              @click="showNoteForm = true; editingNote = null"
            >
              + Add Note
            </button>
          </div>

          <div
            v-if="showNoteForm"
            class="mb-3 bg-gray-800 border border-gray-700 rounded-lg p-4"
          >
            <NoteForm
              :note="editingNote"
              :mode="editingNote ? 'edit' : 'create'"
              :task-id="taskId"
              :locked-context-id="editingNote ? undefined : task.contextId"
              @submit="handleNoteSubmit"
              @cancel="handleNoteCancel"
            />
          </div>

          <NoteList
            :notes="notes"
            :loading="notesLoading"
            @delete="handleDeleteNote"
            @edit="handleEditNote"
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

        <!-- Activity Thread -->
        <div class="mt-6">
          <ThreadPanel
            subject-type="task"
            :subject-id="taskId"
          />
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

    <TaskDebriefDialog
      :open="showDebrief"
      :task-id="taskId"
      @close="showDebrief = false"
    />
  </div>
</template>
