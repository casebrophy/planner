<script setup lang="ts">
import { computed } from 'vue'
import type { DailyPlanItem } from '@/types/dailyPlan'
import type { Task } from '@/types'
import { PriorityColors } from '@/types/enums'

const props = defineProps<{
  item: DailyPlanItem
  task?: Task
}>()

const emit = defineEmits<{
  complete: [id: string]
  dismiss: [id: string]
  click: [id: string]
}>()

const isCompleted = computed(() => props.item.status === 'completed')
</script>

<template>
  <div
    class="flex items-center gap-3 p-3 rounded-lg bg-gray-800 border border-gray-700 hover:border-gray-600 transition-colors cursor-pointer"
    :class="{ 'opacity-50': isCompleted }"
    @click="emit('click', item.id)"
  >
    <!-- Drag handle -->
    <div class="cursor-grab text-gray-500 hover:text-gray-400">
      <svg class="w-4 h-4" fill="currentColor" viewBox="0 0 16 16">
        <path d="M2 4h12M2 8h12M2 12h12" stroke="currentColor" stroke-width="1.5" fill="none" />
      </svg>
    </div>

    <!-- Priority indicator -->
    <div
      v-if="task"
      class="w-1.5 h-8 rounded-full flex-shrink-0"
      :style="{ backgroundColor: PriorityColors[task.priority] || '#6b7280' }"
    />

    <!-- Content -->
    <div class="flex-1 min-w-0">
      <div
        class="text-sm font-medium text-gray-200 truncate"
        :class="{ 'line-through': isCompleted }"
      >
        {{ task?.title || 'Unknown task' }}
      </div>
      <div
        v-if="item.aiPriorityReason"
        class="text-xs text-gray-500 mt-0.5 truncate"
      >
        {{ item.aiPriorityReason }}
      </div>
    </div>

    <!-- Duration badge -->
    <div
      v-if="item.userDurationMin || item.aiDurationMin"
      class="flex-shrink-0"
    >
      <span class="text-xs px-2 py-0.5 rounded-full bg-gray-700 text-gray-300">
        {{ item.userDurationMin || item.aiDurationMin }}m
      </span>
    </div>

    <!-- Actions -->
    <div class="flex gap-1 flex-shrink-0">
      <button
        v-if="!isCompleted"
        class="p-1.5 rounded hover:bg-green-900/50 text-gray-400 hover:text-green-400 transition-colors"
        title="Complete"
        @click.stop="emit('complete', item.id)"
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
            d="M5 13l4 4L19 7"
          />
        </svg>
      </button>
      <button
        v-if="!isCompleted"
        class="p-1.5 rounded hover:bg-red-900/50 text-gray-400 hover:text-red-400 transition-colors"
        title="Dismiss"
        @click.stop="emit('dismiss', item.id)"
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
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      </button>
    </div>
  </div>
</template>
