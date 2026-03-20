<script setup lang="ts">
import { computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import type { Task } from '@/types'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PriorityIndicator from '@/components/shared/PriorityIndicator.vue'
import EnergyIndicator from '@/components/shared/EnergyIndicator.vue'

const props = defineProps<{
  task: Task
}>()

const emit = defineEmits<{
  click: [id: string]
}>()

const dueLabel = computed(() => {
  if (!props.task.dueDate) return null
  return formatDistanceToNow(new Date(props.task.dueDate), { addSuffix: true })
})

const isOverdue = computed(() => {
  if (!props.task.dueDate) return false
  return new Date(props.task.dueDate) < new Date() && props.task.status !== 'done' && props.task.status !== 'cancelled'
})
</script>

<template>
  <div
    class="bg-gray-900 border border-gray-800 rounded-lg p-4 hover:border-gray-700 cursor-pointer transition-colors"
    @click="emit('click', task.id)"
  >
    <div class="flex items-start justify-between gap-3">
      <h3 class="text-sm font-medium text-gray-100 line-clamp-2">
        {{ task.title }}
      </h3>
      <StatusBadge
        :status="task.status"
        type="task"
      />
    </div>

    <p
      v-if="task.description"
      class="mt-1.5 text-xs text-gray-500 line-clamp-2"
    >
      {{ task.description }}
    </p>

    <div class="mt-3 flex items-center gap-3 flex-wrap">
      <PriorityIndicator :priority="task.priority" />
      <EnergyIndicator :energy="task.energy" />
      <span
        v-if="dueLabel"
        class="text-xs"
        :class="isOverdue ? 'text-red-400' : 'text-gray-500'"
      >
        Due {{ dueLabel }}
      </span>
    </div>
  </div>
</template>
