<script setup lang="ts">
import { useEventBoard } from '@/composables/useEventBoard'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import CalendarEventCard from '@/components/calendar-events/CalendarEventCard.vue'
import CalendarEventForm from '@/components/calendar-events/CalendarEventForm.vue'
import type { NewCalendarEvent, UpdateCalendarEvent } from '@/types/calendarEvent'

const {
  loading,
  upcomingEvents,
  pastEvents,
  showForm,
  editingEvent,
  openCreate,
  openEdit,
  closeForm,
  refresh,
  create,
  update,
  remove,
} = useEventBoard()

async function handleSave(data: NewCalendarEvent | UpdateCalendarEvent) {
  if (editingEvent.value) {
    await update(editingEvent.value.id, data as UpdateCalendarEvent)
  } else {
    await create(data as NewCalendarEvent)
  }
  closeForm()
  refresh()
}

async function handleDelete(id: string) {
  await remove(id)
  refresh()
}
</script>

<template>
  <div>
    <PageHeader
      title="Events"
      subtitle="Appointments, trips, and commitments"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
          @click="openCreate"
        >
          Add Event
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading && upcomingEvents.length === 0 && pastEvents.length === 0" />

    <div
      v-else
      class="p-6 space-y-6 max-w-3xl"
    >
      <!-- Upcoming -->
      <div v-if="upcomingEvents.length > 0">
        <h2 class="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
          Upcoming
        </h2>
        <div class="space-y-2">
          <CalendarEventCard
            v-for="event in upcomingEvents"
            :key="event.id"
            :event="event"
            @click="openEdit"
            @delete="handleDelete"
          />
        </div>
      </div>

      <!-- Past -->
      <div v-if="pastEvents.length > 0">
        <h2 class="text-xs font-semibold text-gray-400 uppercase tracking-wide mb-3">
          Past
        </h2>
        <div class="space-y-2">
          <CalendarEventCard
            v-for="event in pastEvents"
            :key="event.id"
            :event="event"
            @click="openEdit"
            @delete="handleDelete"
          />
        </div>
      </div>

      <!-- Empty -->
      <EmptyState
        v-if="upcomingEvents.length === 0 && pastEvents.length === 0"
        title="No events"
        message="Add appointments, trips, and commitments"
        action-label="Add Event"
        @action="openCreate"
      />
    </div>

    <!-- Event Form Drawer -->
    <DrawerPanel
      :open="showForm"
      :title="editingEvent ? 'Edit Event' : 'New Event'"
      @close="closeForm"
    >
      <CalendarEventForm
        :event="editingEvent"
        @save="handleSave"
        @cancel="closeForm"
      />
    </DrawerPanel>
  </div>
</template>
