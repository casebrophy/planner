<script setup lang="ts">
import { computed } from 'vue'
import { format, isSameDay, parseISO, isToday } from 'date-fns'
import CalendarItem from './CalendarItem.vue'
import type { ScheduleItem } from '@/types/schedule'

const props = defineProps<{
  days: Date[]
  timedItems: ScheduleItem[]
  allDayItems: ScheduleItem[]
}>()

const emit = defineEmits<{
  slotClick: [startsAt: string, endsAt: string]
  itemClick: [item: ScheduleItem]
}>()

const hours = Array.from({ length: 17 }, (_, i) => i + 6) // 6am to 10pm

function itemsForDayHour(day: Date, hour: number): ScheduleItem[] {
  return props.timedItems.filter((item) => {
    const start = parseISO(item.startsAt)
    return isSameDay(start, day) && start.getHours() === hour
  })
}

function allDayForDay(day: Date): ScheduleItem[] {
  return props.allDayItems.filter((item) => isSameDay(parseISO(item.startsAt), day))
}

function topPercent(item: ScheduleItem): number {
  const d = parseISO(item.startsAt)
  return (d.getMinutes() / 60) * 100
}

function heightPercent(item: ScheduleItem): number {
  const start = parseISO(item.startsAt)
  const end = parseISO(item.endsAt)
  const durationMin = (end.getTime() - start.getTime()) / 60000
  return Math.max((durationMin / 60) * 100, 25) // min 25% of hour cell
}

function handleSlotClick(day: Date, hour: number) {
  const startsAt = new Date(day)
  startsAt.setHours(hour, 0, 0, 0)
  const endsAt = new Date(startsAt)
  endsAt.setHours(hour + 1, 0, 0, 0)
  emit('slotClick', startsAt.toISOString(), endsAt.toISOString())
}

const hasAllDayItems = computed(() => props.allDayItems.length > 0)
</script>

<template>
  <div class="flex flex-col h-full overflow-auto">
    <!-- Header row: day names -->
    <div class="flex border-b border-gray-800 sticky top-0 bg-gray-900 z-10">
      <div class="w-16 shrink-0" />
      <div
        v-for="day in days"
        :key="day.toISOString()"
        class="flex-1 text-center py-2 border-l border-gray-800"
      >
        <div class="text-xs text-gray-400 uppercase">{{ format(day, 'EEE') }}</div>
        <div
          class="text-sm font-semibold"
          :class="isToday(day) ? 'text-blue-400' : 'text-gray-200'"
        >
          {{ format(day, 'd') }}
        </div>
      </div>
    </div>

    <!-- All-day row -->
    <div
      v-if="hasAllDayItems"
      class="flex border-b border-gray-800"
    >
      <div class="w-16 shrink-0 text-[10px] text-gray-500 text-right pr-2 pt-1">All day</div>
      <div
        v-for="day in days"
        :key="'ad-' + day.toISOString()"
        class="flex-1 border-l border-gray-800 p-1 min-h-[28px]"
      >
        <CalendarItem
          v-for="item in allDayForDay(day)"
          :key="item.id"
          :item="item"
          class="mb-0.5"
          @click="emit('itemClick', item)"
        />
      </div>
    </div>

    <!-- Time grid -->
    <div class="flex flex-1">
      <!-- Hour labels -->
      <div class="w-16 shrink-0">
        <div
          v-for="hour in hours"
          :key="hour"
          class="h-16 border-b border-gray-800/50 text-[10px] text-gray-500 text-right pr-2 pt-0.5"
        >
          {{ hour === 0 ? '12 AM' : hour < 12 ? `${hour} AM` : hour === 12 ? '12 PM' : `${hour - 12} PM` }}
        </div>
      </div>

      <!-- Day columns -->
      <div
        v-for="day in days"
        :key="'col-' + day.toISOString()"
        class="flex-1 border-l border-gray-800"
      >
        <div
          v-for="hour in hours"
          :key="hour"
          class="h-16 border-b border-gray-800/50 relative cursor-pointer hover:bg-gray-800/30"
          @click="handleSlotClick(day, hour)"
        >
          <div
            v-for="item in itemsForDayHour(day, hour)"
            :key="item.id"
            class="absolute left-0.5 right-0.5 z-10"
            :style="{ top: topPercent(item) + '%', height: heightPercent(item) + '%' }"
            @click.stop="emit('itemClick', item)"
          >
            <CalendarItem
              :item="item"
              class="h-full"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
