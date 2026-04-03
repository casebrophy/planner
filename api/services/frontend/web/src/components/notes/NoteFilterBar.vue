<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import type { NoteFilter } from '@/types'
import { useContextStore } from '@/stores/contextStore'

const props = defineProps<{
  filter: NoteFilter
}>()

const emit = defineEmits<{
  update: [filter: NoteFilter]
}>()

const contextStore = useContextStore()

const source = ref<string>(props.filter.source ?? '')
const search = ref<string>(props.filter.search ?? '')
const contextId = ref<string>(props.filter.contextId ?? '')

onMounted(() => {
  contextStore.fetchList()
})

watch([source, search, contextId], () => {
  emit('update', {
    ...props.filter,
    source: source.value || undefined,
    search: search.value || undefined,
    contextId: contextId.value || undefined,
  })
})

function clear() {
  source.value = ''
  search.value = ''
  contextId.value = ''
  emit('update', {})
}
</script>

<template>
  <div class="flex items-center gap-3 flex-wrap">
    <input
      v-model="search"
      type="text"
      placeholder="Search notes..."
      class="bg-gray-800 border border-gray-700 text-gray-100 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >

    <select
      v-model="source"
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >
      <option value="">
        All sources
      </option>
      <option value="manual">
        Manual
      </option>
      <option value="voice">
        Voice
      </option>
      <option value="email">
        Email
      </option>
    </select>

    <select
      v-model="contextId"
      class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
    >
      <option value="">
        All contexts
      </option>
      <option
        v-for="ctx in contextStore.items"
        :key="ctx.id"
        :value="ctx.id"
      >
        {{ ctx.title }}
      </option>
    </select>

    <button
      v-if="source || search || contextId"
      class="text-sm text-gray-400 hover:text-gray-200 transition-colors"
      @click="clear"
    >
      Clear filters
    </button>
  </div>
</template>
