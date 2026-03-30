<script setup lang="ts">
import { ref, computed } from 'vue'
import { ContextStatus } from '@/types/enums'
import type { Context, NewContext, UpdateContext } from '@/types'

const props = defineProps<{
  context?: Context | null
  mode: 'create' | 'edit'
}>()

const emit = defineEmits<{
  submit: [data: NewContext | UpdateContext]
  cancel: []
}>()

const title = ref(props.context?.title ?? '')
const description = ref(props.context?.description ?? '')
const status = ref(props.context?.status ?? ContextStatus.Active)
const summary = ref(props.context?.summary ?? '')

const isValid = computed(() => title.value.trim().length > 0)

function handleSubmit() {
  if (!isValid.value) return

  if (props.mode === 'create') {
    emit('submit', {
      title: title.value.trim(),
      description: description.value.trim(),
    } satisfies NewContext)
  } else {
    emit('submit', {
      title: title.value.trim(),
      description: description.value.trim(),
      status: status.value,
      summary: summary.value.trim(),
    } satisfies UpdateContext)
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
        placeholder="Context title"
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

    <div v-if="mode === 'edit'">
      <label class="block text-sm font-medium text-gray-300 mb-1">Status</label>
      <select
        v-model="status"
        class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
      >
        <option :value="ContextStatus.Active">
          Active
        </option>
        <option :value="ContextStatus.Paused">
          Paused
        </option>
        <option :value="ContextStatus.Closed">
          Closed
        </option>
      </select>
    </div>

    <div v-if="mode === 'edit'">
      <label class="block text-sm font-medium text-gray-300 mb-1">Summary</label>
      <textarea
        v-model="summary"
        rows="2"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        placeholder="High-level summary"
      />
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
        {{ mode === 'create' ? 'Create Context' : 'Save Changes' }}
      </button>
    </div>
  </form>
</template>
