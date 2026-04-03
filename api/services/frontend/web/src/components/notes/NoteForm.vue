<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import type { Note, NewNote, UpdateNote } from '@/types'
import { useContextStore } from '@/stores/contextStore'

const props = defineProps<{
  note?: Note | null
  mode: 'create' | 'edit'
}>()

const emit = defineEmits<{
  submit: [data: NewNote | UpdateNote]
  cancel: []
}>()

const contextStore = useContextStore()

const content = ref(props.note?.content ?? '')
const source = ref(props.note?.source ?? 'manual')
const contextId = ref(props.note?.contextId ?? '')

const isValid = computed(() => content.value.trim().length > 0)

onMounted(() => {
  contextStore.fetchList()
})

function handleSubmit() {
  if (!isValid.value) return

  if (props.mode === 'create') {
    const data: NewNote = {
      content: content.value.trim(),
      source: source.value,
    }
    if (contextId.value) data.contextId = contextId.value
    emit('submit', data)
  } else {
    const data: UpdateNote = {
      content: content.value.trim(),
      source: source.value,
    }
    if (contextId.value) data.contextId = contextId.value
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
      <label class="block text-sm font-medium text-gray-300 mb-1">Content</label>
      <textarea
        v-model="content"
        rows="5"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        placeholder="Note content"
      />
    </div>

    <div class="grid grid-cols-2 gap-4">
      <div>
        <label class="block text-sm font-medium text-gray-300 mb-1">Source</label>
        <select
          v-model="source"
          class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
        >
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
        {{ mode === 'create' ? 'Create Note' : 'Save Changes' }}
      </button>
    </div>
  </form>
</template>
