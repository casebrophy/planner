<script setup lang="ts">
import { computed, ref } from 'vue'
import type { Task } from '@/types'
import { useActivityLogStore } from '@/stores/activityLogStore'

const props = defineProps<{
  habit: Task
  completedDates: string[]
  days: Date[]
}>()

const activityLogStore = useActivityLogStore()
const loggingDate = ref<string | null>(null)

function toDateStr(d: Date): string {
  return d.toISOString().slice(0, 10)
}

function isCompleted(day: Date): boolean {
  return props.completedDates.includes(toDateStr(day))
}

async function toggleDay(day: Date) {
  if (isCompleted(day)) return
  const dateStr = toDateStr(day)
  loggingDate.value = dateStr
  try {
    await activityLogStore.logActivity('task', props.habit.id, dateStr)
  } finally {
    loggingDate.value = null
  }
}

const currentStreak = computed(() => {
  if (props.completedDates.length === 0) return 0
  const sorted = [...props.completedDates].sort().reverse()
  const today = toDateStr(new Date())
  const yesterday = toDateStr(new Date(Date.now() - 86400000))
  if (sorted[0] !== today && sorted[0] !== yesterday) return 0
  let streak = 1
  for (let i = 1; i < sorted.length; i++) {
    const prev = new Date(sorted[i - 1]!)
    const curr = new Date(sorted[i]!)
    const diffDays = (prev.getTime() - curr.getTime()) / 86400000
    if (diffDays === 1) streak++
    else break
  }
  return streak
})
</script>

<template>
  <div class="contents">
    <div class="flex items-center gap-2 px-3 py-2 min-w-0">
      <span class="text-sm text-gray-200 truncate">{{ habit.title }}</span>
      <span v-if="currentStreak > 0" class="text-xs text-purple-400 font-medium whitespace-nowrap">
        {{ currentStreak }}d
      </span>
    </div>
    <div
      v-for="day in days"
      :key="day.toISOString()"
      class="flex items-center justify-center py-2 cursor-pointer"
      @click="toggleDay(day)"
    >
      <div
        class="w-5 h-5 rounded-sm transition-colors"
        :class="[
          isCompleted(day)
            ? 'bg-purple-500'
            : loggingDate === toDateStr(day)
              ? 'bg-purple-300 animate-pulse'
              : 'bg-gray-700 hover:bg-gray-600',
        ]"
      />
    </div>
  </div>
</template>
