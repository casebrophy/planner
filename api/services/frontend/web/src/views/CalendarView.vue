<script setup lang="ts">
import { useCalendar } from '@/composables/useCalendar'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import WeekGrid from '@/components/calendar/WeekGrid.vue'
import TimeBlockForm from '@/components/calendar/TimeBlockForm.vue'
import type { ScheduleItem } from '@/types/schedule'
import type { NewTimeBlock } from '@/types/timeBlock'

const {
  weekDays,
  weekLabel,
  timedItems,
  allDayItems,
  loading,
  nextWeek,
  prevWeek,
  goToToday,
  showForm,
  formStartsAt,
  formEndsAt,
  openCreateAtSlot,
  closeForm,
  createTimeBlock,
  confirmTimeBlock,
} = useCalendar()

function handleSlotClick(startsAt: string, endsAt: string) {
  openCreateAtSlot(startsAt, endsAt)
}

function handleItemClick(item: ScheduleItem) {
  if (item.type === 'time_block' && !item.confirmed) {
    confirmTimeBlock(item.id)
  }
}

async function handleSave(data: NewTimeBlock) {
  await createTimeBlock(data)
  closeForm()
}
</script>

<template>
  <div class="flex flex-col h-full">
    <PageHeader
      title="Calendar"
      :subtitle="weekLabel"
    >
      <template #actions>
        <div class="flex items-center gap-2">
          <button
            class="px-2 py-1 text-sm text-gray-400 hover:text-gray-100 transition-colors"
            @click="prevWeek"
          >
            <svg
              class="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            ><path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M15 19l-7-7 7-7"
            /></svg>
          </button>
          <button
            class="px-3 py-1 text-sm font-medium text-gray-300 hover:text-gray-100 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
            @click="goToToday"
          >
            Today
          </button>
          <button
            class="px-2 py-1 text-sm text-gray-400 hover:text-gray-100 transition-colors"
            @click="nextWeek"
          >
            <svg
              class="w-5 h-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            ><path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M9 5l7 7-7 7"
            /></svg>
          </button>
        </div>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading && timedItems.length === 0" />

    <div
      v-else
      class="flex-1 overflow-hidden"
    >
      <WeekGrid
        :days="weekDays"
        :timed-items="timedItems"
        :all-day-items="allDayItems"
        @slot-click="handleSlotClick"
        @item-click="handleItemClick"
      />
    </div>

    <!-- Time Block Form Drawer -->
    <DrawerPanel
      :open="showForm"
      title="Schedule Task"
      @close="closeForm"
    >
      <TimeBlockForm
        :starts-at="formStartsAt"
        :ends-at="formEndsAt"
        @save="handleSave"
        @cancel="closeForm"
      />
    </DrawerPanel>
  </div>
</template>
