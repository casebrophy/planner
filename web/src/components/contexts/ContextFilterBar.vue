<script setup lang="ts">
import { ref, watch } from 'vue'
import { ContextStatus } from '@/types/enums'
import type { ContextFilter } from '@/types'

const props = defineProps<{
  filter: ContextFilter
}>()

const emit = defineEmits<{
  update: [filter: ContextFilter]
}>()

const status = ref<string>(props.filter.status ?? '')
const title = ref<string>(props.filter.title ?? '')

watch([status, title], () => {
  emit('update', {
    status: status.value ? (status.value as ContextFilter['status']) : undefined,
    title: title.value || undefined,
  })
})

function clear() {
  status.value = ''
  title.value = ''
  emit('update', {})
}
</script>

<template>
  <div class="flex items-center gap-3 flex-wrap">
    <input
      v-model="title"
      type="text"
      placeholder="Search contexts..."
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 w-48 focus:outline-none focus:border-blue-500"
    />

    <select
      v-model="status"
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >
      <option value="">All statuses</option>
      <option :value="ContextStatus.Active">Active</option>
      <option :value="ContextStatus.Paused">Paused</option>
      <option :value="ContextStatus.Closed">Closed</option>
    </select>

    <button
      v-if="status || title"
      class="text-sm text-gray-400 hover:text-gray-200 transition-colors"
      @click="clear"
    >
      Clear filters
    </button>
  </div>
</template>
