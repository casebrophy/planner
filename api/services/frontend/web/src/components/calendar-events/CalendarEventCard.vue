<script setup lang="ts">
import { computed } from 'vue'
import type { CalendarEvent } from '@/types/calendarEvent'

const props = defineProps<{
  event: CalendarEvent
}>()

const emit = defineEmits<{
  click: [event: CalendarEvent]
  delete: [id: string]
}>()

const dateDisplay = computed(() => {
  const start = new Date(props.event.startsAt)
  const end = new Date(props.event.endsAt)

  if (props.event.allDay) {
    return start.toLocaleDateString(undefined, {
      weekday: 'short',
      month: 'short',
      day: 'numeric',
    })
  }

  const dateStr = start.toLocaleDateString(undefined, {
    weekday: 'short',
    month: 'short',
    day: 'numeric',
  })
  const startTime = start.toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  })
  const endTime = end.toLocaleTimeString(undefined, {
    hour: 'numeric',
    minute: '2-digit',
  })
  return `${dateStr} · ${startTime} – ${endTime}`
})

const isPast = computed(() => new Date(props.event.endsAt) < new Date())
</script>

<template>
  <div
    class="flex items-center gap-3 p-4 rounded-lg bg-gray-800 border border-gray-700 hover:border-gray-600 cursor-pointer transition-colors"
    :class="{ 'opacity-50': isPast }"
    @click="emit('click', event)"
  >
    <div class="w-1.5 h-10 rounded-full bg-blue-500 flex-shrink-0" />
    <div class="flex-1 min-w-0">
      <div class="text-sm font-medium text-gray-200">{{ event.title }}</div>
      <div class="text-xs text-gray-400 mt-0.5">{{ dateDisplay }}</div>
      <div
        v-if="event.description"
        class="text-xs text-gray-500 mt-0.5 truncate"
      >{{ event.description }}</div>
    </div>
    <div
      v-if="event.location"
      class="flex-shrink-0"
    >
      <span class="text-xs px-2 py-0.5 rounded-full bg-gray-700 text-gray-300">{{ event.location }}</span>
    </div>
    <div
      v-if="event.allDay"
      class="flex-shrink-0"
    >
      <span class="text-xs px-2 py-0.5 rounded-full bg-blue-900/50 text-blue-300">All day</span>
    </div>
    <button
      class="p-1.5 rounded hover:bg-red-900/50 text-gray-500 hover:text-red-400 transition-colors flex-shrink-0"
      title="Delete"
      @click.stop="emit('delete', event.id)"
    >
      <svg
        class="w-4 h-4"
        fill="none"
        viewBox="0 0 24 24"
        stroke="currentColor"
        stroke-width="2"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
        />
      </svg>
    </button>
  </div>
</template>
