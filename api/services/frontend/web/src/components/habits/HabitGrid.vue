<script setup lang="ts">
import { computed } from 'vue'
import type { Task } from '@/types'
import HabitRow from './HabitRow.vue'

const props = defineProps<{
  habits: Task[]
  habitGrid: Record<string, string[]>
  days: Date[]
}>()

const dayLabels = computed(() =>
  props.days.map((d) => {
    const day = d.getDate()
    const month = d.toLocaleDateString('en-US', { month: 'short' })
    return { date: d, label: `${month} ${day}` }
  }),
)

function getCompletedDates(habitId: string): string[] {
  return props.habitGrid[habitId] ?? []
}
</script>

<template>
  <div v-if="habits.length === 0" class="text-center py-12 text-gray-500">
    <p class="text-lg">No habits found</p>
    <p class="text-sm mt-1">Create a task with a recurrence rule to track it as a habit.</p>
  </div>

  <div v-else class="overflow-x-auto">
    <div
      class="grid gap-0 min-w-max"
      :style="{ gridTemplateColumns: `180px repeat(${days.length}, 1fr)` }"
    >
      <!-- Header row -->
      <div class="px-3 py-2 text-xs font-medium text-gray-500 border-b border-gray-700">
        Habit
      </div>
      <div
        v-for="{ date, label } in dayLabels"
        :key="date.toISOString()"
        class="text-center text-xs text-gray-500 py-2 border-b border-gray-700"
      >
        {{ label }}
      </div>

      <!-- Habit rows -->
      <HabitRow
        v-for="habit in habits"
        :key="habit.id"
        :habit="habit"
        :completed-dates="getCompletedDates(habit.id)"
        :days="days"
      />
    </div>
  </div>
</template>
