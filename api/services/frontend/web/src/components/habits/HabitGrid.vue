<script setup lang="ts">
import type { Task } from '@/types'
import type { HabitGridMap } from '@/stores/activityLogStore'

defineProps<{
  habits: Task[]
  habitGrid: HabitGridMap
  days: Date[]
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
  return !!row[dateKey(day)]
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
          <th class="text-left py-2 px-3 text-gray-400 font-medium sticky left-0 bg-gray-900 min-w-[180px]">
            Habit
          </th>
          <th
            v-for="day in days"
            :key="dateKey(day)"
            class="py-2 px-1 text-gray-500 font-normal text-center min-w-[32px]"
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
          <td class="py-2 px-3 text-gray-200 sticky left-0 bg-gray-900 truncate max-w-[180px]">
            {{ habit.title }}
          </td>
          <td
            v-for="day in days"
            :key="dateKey(day)"
            class="py-2 px-1 text-center"
          >
            <div
              class="w-5 h-5 mx-auto rounded-sm"
              :class="isCompleted(habit.id, day, habitGrid)
                ? 'bg-purple-500'
                : 'bg-gray-800'"
            />
          </td>
        </tr>
      </tbody>
    </table>
  </div>
</template>
