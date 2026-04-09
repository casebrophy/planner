<script setup lang="ts">
import { computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import type { Note } from '@/types'

const props = defineProps<{
  note: Note
}>()

const emit = defineEmits<{
  click: [id: string]
}>()

const sourceColors: Record<string, string> = {
  manual: '#6b7280',
  voice: '#8b5cf6',
  email: '#f59e0b',
}

const timeLabel = computed(() =>
  formatDistanceToNow(new Date(props.note.createdAt), { addSuffix: true })
)
</script>

<template>
  <div
    class="bg-gray-900 border border-gray-800 rounded-lg p-4 hover:border-gray-700 cursor-pointer transition-colors"
    @click="emit('click', note.id)"
  >
    <p class="text-sm text-gray-100 line-clamp-3">
      {{ note.content }}
    </p>
    <div class="mt-3 flex items-center gap-3">
      <span
        v-if="note.unconfirmed"
        class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-900/30 text-amber-400 border border-amber-700/40"
      >
        Unconfirmed
      </span>
      <span
        class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium"
        :style="{ backgroundColor: (sourceColors[note.source] ?? '#6b7280') + '20', color: sourceColors[note.source] ?? '#6b7280' }"
      >
        {{ note.source }}
      </span>
      <span class="text-xs text-gray-500">{{ timeLabel }}</span>
    </div>
  </div>
</template>
