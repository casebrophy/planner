<script setup lang="ts">
import { computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import type { ContextEvent } from '@/types'

const props = defineProps<{
  event: ContextEvent
}>()

const timeAgo = computed(() =>
  formatDistanceToNow(new Date(props.event.createdAt), { addSuffix: true }),
)

const kindClass = computed(() => {
  switch (props.event.kind) {
    case 'note':
      return 'bg-blue-500'
    case 'status_change':
      return 'bg-amber-500'
    default:
      return 'bg-gray-500'
  }
})
</script>

<template>
  <div class="relative pl-9">
    <!-- Dot -->
    <div
      class="absolute left-2 top-1.5 w-3 h-3 rounded-full border-2 border-gray-950"
      :class="kindClass"
    />

    <div class="bg-gray-900 border border-gray-800 rounded-lg p-3">
      <div class="flex items-center justify-between mb-1">
        <span class="text-xs font-medium text-gray-300 uppercase tracking-wider">
          {{ event.kind }}
        </span>
        <span class="text-xs text-gray-500">{{ timeAgo }}</span>
      </div>
      <p class="text-sm text-gray-300 whitespace-pre-wrap">{{ event.content }}</p>
    </div>
  </div>
</template>
