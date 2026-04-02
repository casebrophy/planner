<script setup lang="ts">
import type { ScheduleItem } from '@/types/schedule'

defineProps<{
  item: ScheduleItem
}>()

defineEmits<{
  click: [item: ScheduleItem]
}>()
</script>

<template>
  <div
    class="px-2 py-1 rounded text-xs cursor-pointer truncate border-l-2"
    :class="[
      item.type === 'event'
        ? 'bg-blue-900/40 border-blue-500 text-blue-200 hover:bg-blue-900/60'
        : item.confirmed
          ? 'bg-green-900/40 border-green-500 text-green-200 hover:bg-green-900/60'
          : 'bg-gray-800 border-gray-500 text-gray-300 hover:bg-gray-700',
    ]"
    @click="$emit('click', item)"
  >
    <div class="flex items-center gap-1">
      <span
        v-if="item.type === 'time_block' && !item.confirmed"
        class="text-gray-500"
        title="Unconfirmed"
      >?</span>
      <span class="truncate font-medium">{{ item.title || 'Task block' }}</span>
    </div>
    <div
      v-if="!item.allDay"
      class="text-[10px] opacity-70"
    >
      {{ formatTime(item.startsAt) }} – {{ formatTime(item.endsAt) }}
    </div>
  </div>
</template>

<script lang="ts">
function formatTime(iso: string): string {
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour: 'numeric', minute: '2-digit' })
}
</script>
