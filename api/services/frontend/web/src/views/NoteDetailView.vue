<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useNoteDetail } from '@/composables/useNoteDetail'
import { useTagStore } from '@/stores/tagStore'
import NoteForm from '@/components/notes/NoteForm.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import type { UpdateNote } from '@/types'

const route = useRoute()
const router = useRouter()
const noteId = route.params.id as string

const { note, tags, loading, update, remove, addTag, removeTag } = useNoteDetail(noteId)
const tagStore = useTagStore()

const editing = ref(false)
const confirmDelete = ref(false)

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
              class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="confirmDelete = true"
            >
              Delete
            </button>
          </div>
        </div>

        <p class="text-sm text-gray-100 whitespace-pre-wrap">{{ note.content }}</p>

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
            <span class="text-gray-300">{{ note.contextId }}</span>
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
          <h4 class="text-sm font-medium text-gray-300 mb-2">Tags</h4>
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

        <!-- Activity Thread -->
        <div class="mt-6">
          <ThreadPanel
            subject-type="note"
            :subject-id="noteId"
          />
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
  </div>
</template>
