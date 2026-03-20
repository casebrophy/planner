<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { TaskPriority, TaskEnergy, TaskStatus } from '@/types/enums'
import type { Task, NewTask, UpdateTask } from '@/types'
import { useContextStore } from '@/stores/contextStore'

const props = defineProps<{
  task?: Task | null
  mode: 'create' | 'edit'
}>()

const emit = defineEmits<{
  submit: [data: NewTask | UpdateTask]
  cancel: []
}>()

const contextStore = useContextStore()

const title = ref(props.task?.title ?? '')
const description = ref(props.task?.description ?? '')
const status = ref(props.task?.status ?? TaskStatus.Todo)
const priority = ref(props.task?.priority ?? TaskPriority.Medium)
const energy = ref(props.task?.energy ?? TaskEnergy.Medium)
const contextId = ref(props.task?.contextId ?? '')
const dueDate = ref(props.task?.dueDate ? props.task.dueDate.slice(0, 16) : '')

const isValid = computed(() => title.value.trim().length > 0)

onMounted(() => {
  contextStore.fetchContexts()
})

function handleSubmit() {
  if (!isValid.value) return

  if (props.mode === 'create') {
    const data: NewTask = {
      title: title.value.trim(),
      description: description.value.trim(),
      priority: priority.value as NewTask['priority'],
      energy: energy.value as NewTask['energy'],
    }
    if (contextId.value) data.contextId = contextId.value
    if (dueDate.value) data.dueDate = new Date(dueDate.value).toISOString()
    emit('submit', data)
  } else {
    const data: UpdateTask = {
      title: title.value.trim(),
      description: description.value.trim(),
      status: status.value as UpdateTask['status'],
      priority: priority.value as UpdateTask['priority'],
      energy: energy.value as UpdateTask['energy'],
    }
    if (contextId.value) data.contextId = contextId.value
    if (dueDate.value) data.dueDate = new Date(dueDate.value).toISOString()
    emit('submit', data)
  }
}
</script>

<template>
  <form
    class="space-y-4"
    @submit.prevent="handleSubmit"
  >
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Title</label>
      <input
        v-model="title"
        type="text"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        placeholder="Task title"
      >
    </div>

    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Description</label>
      <textarea
        v-model="description"
        rows="3"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        placeholder="Description"
      />
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div v-if="mode === 'edit'">
        <label class="block text-sm font-medium text-gray-300 mb-1">Status</label>
        <select
          v-model="status"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
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
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Priority</label>
        <select
          v-model="priority"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
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
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Energy</label>
        <select
          v-model="energy"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
          <option :value="TaskEnergy.Low">
            Low
          </option>
          <option :value="TaskEnergy.Medium">
            Medium
          </option>
          <option :value="TaskEnergy.High">
            High
          </option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Context</label>
        <select
          v-model="contextId"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
          <option value="">
            No context
          </option>
          <option
            v-for="ctx in contextStore.items"
            :key="ctx.id"
            :value="ctx.id"
          >
            {{ ctx.title }}
          </option>
        </select>
      </div>

      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Due Date</label>
        <input
          v-model="dueDate"
          type="datetime-local"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
      </div>
    </div>

    <div class="flex justify-end gap-3 pt-2">
      <button
        type="button"
        class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
        @click="emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        :disabled="!isValid"
        class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        {{ mode === 'create' ? 'Create Task' : 'Save Changes' }}
      </button>
    </div>
  </form>
</template>
