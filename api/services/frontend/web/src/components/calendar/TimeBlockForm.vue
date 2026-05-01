<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { storeToRefs } from 'pinia'
import type { NewTimeBlock } from '@/types/timeBlock'

const props = defineProps<{
  startsAt?: string
  endsAt?: string
}>()

const emit = defineEmits<{
  save: [data: NewTimeBlock]
  cancel: []
}>()

const taskStore = useTaskStore()
const { items: tasks } = storeToRefs(taskStore)

const selectedTaskId = ref('')
const startsAtLocal = ref('')
const endsAtLocal = ref('')

onMounted(async () => {
  await taskStore.fetchAll()
  if (props.startsAt) {
    startsAtLocal.value = toLocalDatetime(props.startsAt)
  }
  if (props.endsAt) {
    endsAtLocal.value = toLocalDatetime(props.endsAt)
  }
})

function toLocalDatetime(iso: string): string {
  const d = new Date(iso)
  const offset = d.getTimezoneOffset()
  const local = new Date(d.getTime() - offset * 60000)
  return local.toISOString().slice(0, 16)
}

function handleSubmit() {
  if (!selectedTaskId.value || !startsAtLocal.value || !endsAtLocal.value) return

  emit('save', {
    taskId: selectedTaskId.value,
    startsAt: new Date(startsAtLocal.value).toISOString(),
    endsAt: new Date(endsAtLocal.value).toISOString(),
  })
}

// Filter to open tasks only
const openTasks = ref<typeof tasks.value>([])
onMounted(() => {
  openTasks.value = tasks.value.filter((t) => t.status === 'open' || t.status === 'blocked')
})
</script>

<template>
  <form
    class="space-y-4"
    @submit.prevent="handleSubmit"
  >
    <!-- Task picker -->
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Task</label>
      <select
        v-model="selectedTaskId"
        class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-gray-100 text-sm focus:outline-none focus:border-blue-500"
      >
        <option
          value=""
          disabled
        >
          Select a task...
        </option>
        <option
          v-for="task in openTasks"
          :key="task.id"
          :value="task.id"
        >
          {{ task.title }}
        </option>
      </select>
    </div>

    <!-- Start time -->
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">Start</label>
      <input
        v-model="startsAtLocal"
        type="datetime-local"
        class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-gray-100 text-sm focus:outline-none focus:border-blue-500"
      >
    </div>

    <!-- End time -->
    <div>
      <label class="block text-sm font-medium text-gray-300 mb-1">End</label>
      <input
        v-model="endsAtLocal"
        type="datetime-local"
        class="w-full bg-gray-800 border border-gray-700 rounded-lg px-3 py-2 text-gray-100 text-sm focus:outline-none focus:border-blue-500"
      >
    </div>

    <!-- Actions -->
    <div class="flex justify-end gap-2 pt-2">
      <button
        type="button"
        class="px-3 py-1.5 text-sm text-gray-400 hover:text-gray-100 transition-colors"
        @click="emit('cancel')"
      >
        Cancel
      </button>
      <button
        type="submit"
        class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
        :disabled="!selectedTaskId || !startsAtLocal || !endsAtLocal"
      >
        Schedule
      </button>
    </div>
  </form>
</template>
