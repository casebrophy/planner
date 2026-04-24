<script setup lang="ts">
import { ref, computed, watchEffect } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNoteDetail } from '@/composables/useNoteDetail'
import { useTagStore } from '@/stores/tagStore'
import { useContextStore } from '@/stores/contextStore'
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import { useNoteStore } from '@/stores/noteStore'
import { useRelatedByContext } from '@/composables/useRelatedByContext'
import NoteForm from '@/components/notes/NoteForm.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import ActivityLogButton from '@/components/shared/ActivityLogButton.vue'
import StreakDisplay from '@/components/shared/StreakDisplay.vue'
import ActivityHistory from '@/components/shared/ActivityHistory.vue'
import { correctionService } from '@/services/correctionService'
import { noteService } from '@/services/noteService'
import type { UpdateNote, EntityLink } from '@/types'

const route = useRoute()
const router = useRouter()
const noteId = route.params.id as string

const { note, tags, loading, update, remove, addTag, removeTag } = useNoteDetail(noteId)
const tagStore = useTagStore()
const contextStore = useContextStore()
contextStore.fetchList()
const entityLinkStore = useEntityLinkStore()
const noteStore = useNoteStore()

const { tasks: sameContextTasks, notes: sameContextNotes } = useRelatedByContext(
  computed(() => note.value?.contextId),
  'note',
  noteId,
)

const hasSameContextItems = computed(() =>
  sameContextTasks.value.length > 0 || sameContextNotes.value.length > 0
)

watchEffect(async () => {
  if (note.value?.id) {
    await entityLinkStore.fetchLinks('note', note.value.id)
  }
})

const explicitLinks = computed(() => {
  if (!note.value?.id) return []
  return entityLinkStore.getLinks('note', note.value.id)
})

const showLinkModal = ref(false)
const linkTargetType = ref<'task' | 'note' | 'event'>('task')
const linkTargetId = ref('')

async function addLink() {
  if (!note.value?.id || !linkTargetId.value.trim()) return
  await entityLinkStore.createLink({
    sourceType: 'note',
    sourceId: note.value.id,
    targetType: linkTargetType.value,
    targetId: linkTargetId.value.trim(),
  })
  showLinkModal.value = false
  linkTargetId.value = ''
}

async function removeLink(link: EntityLink) {
  await entityLinkStore.deleteLink(link)
}

const contextName = computed(() => {
  if (!note.value?.contextId) return null
  return contextStore.items.find(c => c.id === note.value!.contextId)?.title ?? note.value.contextId
})

const editing = ref(false)
const confirmDelete = ref(false)
const showConvertConfirm = ref(false)
const correcting = ref(false)
const converting = ref(false)

async function handlePromote(newType: 'task' | 'event') {
  if (!note.value?.id || correcting.value) return
  correcting.value = true
  try {
    await correctionService.correct(note.value.id, 'note', newType)
    router.push({ name: newType === 'task' ? 'tasks' : 'tasks' })
  } finally {
    correcting.value = false
  }
}

const sourceColors: Record<string, string> = {
  manual: '#6b7280',
  voice: '#8b5cf6',
  email: '#f59e0b',
}

async function handleUpdate(data: UpdateNote | Record<string, unknown>) {
  await update(data as UpdateNote)
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'notes' })
}

async function handleConvertToTask() {
  if (!note.value?.id || converting.value) return
  converting.value = true
  try {
    const newTask = await noteService.convertNoteToTask(note.value.id)
    noteStore.remove(note.value.id)
    showConvertConfirm.value = false
    const navTo: Record<string, unknown> = {
      name: 'task-detail',
      params: { id: newTask.id },
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
      v-if="loading && !note"
      size="sm"
    />

    <div
      v-else-if="note"
      class="space-y-6"
    >
      <!-- View Mode -->
      <div v-if="!editing">
        <!-- Unconfirmed classification banner -->
        <div
          v-if="note.unconfirmed"
          class="mb-4 p-3 rounded-lg bg-amber-900/20 border border-amber-700/40"
        >
          <p class="text-sm text-amber-300 mb-2">
            This item was created with low classifier confidence. Is it really a note?
          </p>
          <div class="flex gap-2">
            <button
              :disabled="correcting"
              class="px-3 py-1.5 text-xs font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40"
              @click="handlePromote('task')"
            >
              Move to Task
            </button>
          </div>
        </div>
        <div class="flex items-center justify-between mb-4">
          <div class="flex items-center gap-2">
            <span
              class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
              :style="{ backgroundColor: (sourceColors[note.source] ?? '#6b7280') + '20', color: sourceColors[note.source] ?? '#6b7280' }"
            >
              {{ note.source }}
            </span>
          </div>
          <div class="flex gap-2">
            <button
              class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="editing = true"
            >
              Edit
            </button>
            <button
              class="px-3 py-1.5 text-sm text-emerald-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="showConvertConfirm = true"
            >
              Convert to Task
            </button>
            <button
              class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="confirmDelete = true"
            >
              Delete
            </button>
          </div>
        </div>

        <p class="text-sm text-gray-100 whitespace-pre-wrap">
          {{ note.content }}
        </p>

        <div class="space-y-3 text-sm mt-4">
          <div class="flex justify-between">
            <span class="text-gray-500">Source</span>
            <span class="text-gray-300">{{ note.source }}</span>
          </div>
          <div
            v-if="note.contextId"
            class="flex justify-between"
          >
            <span class="text-gray-500">Context</span>
            <span class="text-gray-300">{{ contextName }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Created</span>
            <span class="text-gray-300">{{ new Date(note.createdAt).toLocaleString() }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Updated</span>
            <span class="text-gray-300">{{ new Date(note.updatedAt).toLocaleString() }}</span>
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
              subject-type="note"
              :subject-id="noteId"
            />
          </div>
          <StreakDisplay
            subject-type="note"
            :subject-id="noteId"
          />
          <div class="mt-3">
            <ActivityHistory
              subject-type="note"
              :subject-id="noteId"
            />
          </div>
        </div>

        <!-- Activity Thread -->
        <div class="mt-6">
          <ThreadPanel
            subject-type="note"
            :subject-id="noteId"
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
                {{ link.sourceId === note.id ? link.targetType : link.sourceType }}:
                {{ link.sourceId === note.id ? link.targetId : link.sourceId }}
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
      <NoteForm
        v-else
        :note="note"
        mode="edit"
        @submit="handleUpdate"
        @cancel="editing = false"
      />
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete Note"
      message="Are you sure you want to delete this note? This action cannot be undone."
      @confirm="handleDelete"
      @cancel="confirmDelete = false"
    />

    <ConfirmDialog
      :open="showConvertConfirm"
      title="Convert to Task"
      message="Convert this note to a task? Tags and context will be preserved."
      :loading="converting"
      @confirm="handleConvertToTask"
      @cancel="showConvertConfirm = false"
    />
  </div>
</template>
