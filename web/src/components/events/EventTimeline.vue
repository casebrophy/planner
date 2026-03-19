<script setup lang="ts">
import type { ContextEvent } from '@/types'
import EventTimelineItem from './EventTimelineItem.vue'

defineProps<{
  events: ContextEvent[]
}>()
</script>

<template>
  <div class="relative">
    <!-- Vertical line -->
    <div class="absolute left-3.5 top-0 bottom-0 w-px bg-gray-800" />

    <div class="space-y-4">
      <EventTimelineItem v-for="event in events" :key="event.id" :event="event" />
    </div>

    <div v-if="events.length === 0" class="text-center py-8 text-sm text-gray-500">
      No events yet
    </div>
  </div>
</template>
