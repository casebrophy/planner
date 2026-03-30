<script setup lang="ts">
import { ref, watch } from 'vue'
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

watch([status, priority], () => {
  emit('update', {
    ...props.filter,
    status: status.value ? (status.value as TaskFilter['status']) : undefined,
    priority: priority.value ? (priority.value as TaskFilter['priority']) : undefined,
  })
})

function clear() {
  status.value = ''
  priority.value = ''
  emit('update', {})
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
      <option :value="TaskStatus.Todo">
        To Do
      </option>
      <option :value="TaskStatus.InProgress">
        In Progress
      </option>
      <option :value="TaskStatus.Done">
        Done
      </option>
      <option :value="TaskStatus.Cancelled">
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
      v-if="status || priority"
      class="text-sm text-gray-400 hover:text-gray-200 transition-colors"
      @click="clear"
    >
      Clear filters
    </button>
  </div>
</template>
