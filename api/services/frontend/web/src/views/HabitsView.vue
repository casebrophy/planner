<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useActivityLogStore } from '@/stores/activityLogStore'
import HabitGrid from '@/components/habits/HabitGrid.vue'

const taskStore = useTaskStore()
const activityLogStore = useActivityLogStore()

const dayRange = ref(30)
const loading = ref(false)

const days = computed(() => {
  const result: Date[] = []
  const now = new Date()
  for (let i = dayRange.value - 1; i >= 0; i--) {
    const d = new Date(now)
    d.setDate(d.getDate() - i)
    d.setHours(0, 0, 0, 0)
    result.push(d)
  }
  return result
})

async function loadData() {
  loading.value = true
  try {
    await taskStore.fetchHabits()
    const habitIds = taskStore.habits.map((h) => h.id)
    if (habitIds.length > 0) {
      await activityLogStore.fetchHabitGrid(habitIds, dayRange.value)
    }
  } finally {
    loading.value = false
  }
}

onMounted(loadData)
watch(dayRange, loadData)
</script>

<template>
  <div class="space-y-6">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-semibold text-gray-100">
        Habits
      </h1>
      <div class="flex gap-1 bg-gray-800 rounded-lg p-1">
        <button
          v-for="range in [30, 90]"
          :key="range"
          class="px-3 py-1 text-sm rounded-md transition-colors"
          :class="dayRange === range ? 'bg-purple-600 text-white' : 'text-gray-400 hover:text-gray-200'"
          @click="dayRange = range"
        >
          {{ range }}d
        </button>
      </div>
    </div>

    <div
      v-if="loading"
      class="space-y-3"
    >
      <div
        v-for="i in 5"
        :key="i"
        class="h-10 bg-gray-800 rounded animate-pulse"
      />
    </div>

    <HabitGrid
      v-else
      :habits="taskStore.habits"
      :habit-grid="activityLogStore.habitGrid"
      :days="days"
    />
  </div>
</template>
