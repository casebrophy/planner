<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { formatDistanceToNow } from 'date-fns'
import type { Task } from '@/types'
import { useContextStore } from '@/stores/contextStore'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import PriorityIndicator from '@/components/shared/PriorityIndicator.vue'
import EnergyIndicator from '@/components/shared/EnergyIndicator.vue'

const props = defineProps<{
  task: Task
}>()

const emit = defineEmits<{
  click: [id: string]
}>()

const router = useRouter()
const contextStore = useContextStore()

const context = computed(() => {
  if (!props.task.contextId) return null
  return contextStore.contextById(props.task.contextId) ?? null
})

const dueLabel = computed(() => {
  if (!props.task.dueDate) return null
  return formatDistanceToNow(new Date(props.task.dueDate), { addSuffix: true })
})

const isOverdue = computed(() => {
  if (!props.task.dueDate) return false
  return new Date(props.task.dueDate) < new Date() && props.task.status !== 'done' && props.task.status !== 'dismissed'
})

function navigateToContext() {
  if (context.value) {
    router.push(`/contexts/${context.value.id}`)
  }
}
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
      <span
        v-if="task.recurrenceRule"
        class="inline-flex items-center gap-1 text-xs text-blue-400"
        title="Recurring"
      >
        <svg
          class="w-3.5 h-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
      </span>
      <PriorityIndicator :priority="task.priority" />
      <EnergyIndicator :energy="task.energy" />
      <span
        v-if="dueLabel"
        class="text-xs"
        :class="isOverdue ? 'text-red-400' : 'text-gray-500'"
      >
        Due {{ dueLabel }}
      </span>
      <button
        v-if="context"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-400 border border-gray-700 hover:text-gray-200 hover:border-gray-600 transition-colors"
        @click.stop="navigateToContext"
      >
        {{ context.title }}
      </button>
    </div>
  </div>
</template>
