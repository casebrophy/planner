<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { TaskStatus, TaskPriority } from '@/types/enums'
import type { TaskFilter } from '@/types'

const props = defineProps<{
  filter: TaskFilter
}>()

const emit = defineEmits<{
  update: [filter: TaskFilter]
}>()

const status = ref<string>(props.filter.status ?? '')
const priority = ref<string>(props.filter.priority ?? '')
let clearing = false

const showCompleted = computed(() => !props.filter.excludeStatuses?.length)

watch([status, priority], () => {
  if (clearing) return
  const newFilter: TaskFilter = {
    ...props.filter,
    status: status.value ? (status.value as TaskFilter['status']) : undefined,
    priority: priority.value ? (priority.value as TaskFilter['priority']) : undefined,
  }
  // If user explicitly selects done/dismissed, clear the exclusion filter
  if (status.value === TaskStatus.Done || status.value === TaskStatus.Dismissed) {
    newFilter.excludeStatuses = undefined
  }
  emit('update', newFilter)
})

function clear() {
  clearing = true
  status.value = ''
  priority.value = ''
  emit('update', { excludeStatuses: [TaskStatus.Done, TaskStatus.Dismissed] })
  nextTick(() => { clearing = false })
}

function toggleCompleted() {
  const newFilter: TaskFilter = { ...props.filter }
  if (showCompleted.value) {
    newFilter.excludeStatuses = [TaskStatus.Done, TaskStatus.Dismissed]
  } else {
    newFilter.excludeStatuses = undefined
  }
  emit('update', newFilter)
}
</script>

<template>
  <div class="flex items-center gap-3 flex-wrap">
    <select
      v-model="status"
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >
      <option value="">
        All statuses
      </option>
      <option :value="TaskStatus.Open">
        To Do
      </option>
      <option :value="TaskStatus.Blocked">
        In Progress
      </option>
      <option :value="TaskStatus.Done">
        Done
      </option>
      <option :value="TaskStatus.Dismissed">
        Cancelled
      </option>
    </select>

    <select
      v-model="priority"
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >
      <option value="">
        All priorities
      </option>
      <option :value="TaskPriority.Low">
        Low
      </option>
      <option :value="TaskPriority.Medium">
        Medium
      </option>
      <option :value="TaskPriority.High">
        High
      </option>
      <option :value="TaskPriority.Urgent">
        Urgent
      </option>
    </select>

    <button
      class="text-sm px-3 py-1.5 rounded-lg border transition-colors"
      :class="showCompleted
        ? 'bg-gray-700 text-gray-200 border-gray-600 hover:bg-gray-600'
        : 'text-gray-400 bg-gray-800 border-gray-700 hover:bg-gray-750 hover:text-gray-300'"
      @click="toggleCompleted"
    >
      {{ showCompleted ? 'Hide completed' : 'Show completed' }}
    </button>

    <button
      v-if="status || priority"
      class="text-sm text-gray-400 hover:text-gray-200 transition-colors"
      @click="clear"
    >
      Clear filters
    </button>
  </div>
</template>
