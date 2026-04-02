<script setup lang="ts">
import { computed } from 'vue'
import type { ContextEvent } from '@/types/event'

const props = defineProps<{
  event: ContextEvent
}>()

const timeRange = computed(() => {
  // Since ContextEvent doesn't have time fields, we'll display just the content
  // This can be expanded if event types are added later
  return props.event.content
})
</script>

<template>
  <div class="flex items-center gap-3 p-3 rounded-lg bg-blue-900/20 border border-blue-800/50">
    <div class="w-1.5 h-8 rounded-full bg-blue-500 flex-shrink-0" />
    <div class="flex-1 min-w-0">
      <div class="text-sm font-medium text-gray-200">{{ event.kind }}</div>
      <div class="text-xs text-blue-400 mt-0.5 truncate">{{ timeRange }}</div>
    </div>
  </div>
</template>
