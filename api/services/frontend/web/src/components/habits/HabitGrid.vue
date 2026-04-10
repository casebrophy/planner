<script setup lang="ts">
import type { Task } from '@/types'
import type { HabitGridMap } from '@/stores/activityLogStore'

defineProps<{
  habits: Task[]
  habitGrid: HabitGridMap
  days: Date[]
}>()

const emit = defineEmits<{
  toggle: [habitId: string, day: Date]
}>()

function formatDay(d: Date): string {
  return d.toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric' })
}

function dateKey(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function isCompleted(habitId: string, day: Date, grid: HabitGridMap): boolean {
  const row = grid[habitId]
  if (!row) return false
  return row.some((entry) => entry.date === dateKey(day))
}

function isToday(d: Date): boolean {
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}
</script>

<template>
  <div
    v-if="habits.length === 0"
    class="text-center py-12 text-gray-500"
  >
    <p class="text-lg">
      No habits yet
    </p>
    <p class="text-sm mt-1">
      Create a recurring task to track it as a habit.
    </p>
  </div>

  <div
    v-else
    class="overflow-x-auto"
  >
    <table class="w-full text-sm">
      <thead>
        <tr>
          <th class="text-left py-2 px-3 text-gray-400 font-medium sticky left-0 bg-gray-950 min-w-[180px]">
            Habit
          </th>
          <th
            v-for="day in days"
            :key="dateKey(day)"
            class="py-2 px-1 text-gray-500 font-normal text-center min-w-[32px]"
            :class="isToday(day) ? 'text-purple-400' : ''"
            :title="formatDay(day)"
          >
            <span class="text-xs">{{ day.getDate() }}</span>
          </th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="habit in habits"
          :key="habit.id"
          class="border-t border-gray-800"
        >
          <td class="py-2 px-3 text-gray-200 sticky left-0 bg-gray-950">
            <span class="block truncate max-w-[180px]">{{ habit.title }}</span>
          </td>
          <td
            v-for="day in days"
            :key="dateKey(day)"
            class="py-2 px-1 text-center"
          >
            <button
              class="w-5 h-5 mx-auto rounded-sm transition-colors cursor-pointer"
              :class="isCompleted(habit.id, day, habitGrid)
                ? 'bg-purple-500 hover:bg-purple-400'
                : 'bg-gray-800 hover:bg-gray-700'"
              :title="`${habit.title} — ${formatDay(day)}`"
              @click="emit('toggle', habit.id, day)"
            />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
