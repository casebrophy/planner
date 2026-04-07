<script setup lang="ts">
import { defineProps, defineEmits } from 'vue'
import type { Note } from '@/types'

defineProps<{
  notes: Note[]
  loading?: boolean
}>()

const emit = defineEmits<{
  edit: [note: Note]
  delete: [note: Note]
}>()

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString()
}

function truncateContent(content: string, maxLength = 100) {
  return content.length > maxLength ? content.substring(0, maxLength) + '…' : content
}
</script>

<template>
  <div class="space-y-2">
    <div
      v-if="notes.length === 0 && !loading"
      class="text-sm text-gray-500 py-4 text-center"
    >
      No notes yet. Create one to get started.
    </div>

    <div
      v-for="note in notes"
      :key="note.id"
      class="bg-gray-800 border border-gray-700 rounded-lg p-3 hover:border-gray-600 transition-colors"
    >
      <div class="flex items-start justify-between gap-2 mb-2">
        <p class="text-sm text-gray-200 flex-1 break-words">
          {{ truncateContent(note.content) }}
        </p>
        <div class="flex gap-1 flex-shrink-0">
          <button
            class="text-gray-500 hover:text-blue-400 transition-colors p-1"
            aria-label="Edit note"
            @click="emit('edit', note)"
          >
            ✎
          </button>
          <button
            class="text-gray-500 hover:text-red-400 transition-colors p-1"
            aria-label="Delete note"
            @click="emit('delete', note)"
          >
            ×
          </button>
        </div>
      </div>

      <div class="flex items-center justify-between text-xs">
        <span class="text-gray-500">{{ note.source }}</span>
        <span class="text-gray-600">{{ formatDate(note.createdAt) }}</span>
      </div>
    </div>
  </div>
</template>
