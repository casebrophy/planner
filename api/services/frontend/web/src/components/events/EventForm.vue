<script setup lang="ts">
import { ref, computed } from 'vue'
import type { NewEvent } from '@/types'

const emit = defineEmits<{
  submit: [event: NewEvent]
}>()

const kind = ref('note')
const content = ref('')

const isValid = computed(() => kind.value.trim().length > 0 && content.value.trim().length > 0)

function handleSubmit() {
  if (!isValid.value) return
  emit('submit', {
    kind: kind.value.trim(),
    content: content.value.trim(),
  })
  content.value = ''
}
</script>

<template>
  <form
    class="space-y-3"
    @submit.prevent="handleSubmit"
  >
    <div class="flex gap-3">
      <select
        v-model="kind"
        class="bg-gray-800 border border-gray-700 text-gray-300 text-sm rounded-lg px-3 py-2 focus:outline-none focus:border-blue-500"
      >
        <option value="note">
          Note
        </option>
        <option value="status_change">
          Status Change
        </option>
        <option value="update">
          Update
        </option>
        <option value="decision">
          Decision
        </option>
      </select>
    </div>

    <textarea
      v-model="content"
      rows="3"
      class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
      placeholder="Add event content..."
    />

    <div class="flex justify-end">
      <button
        type="submit"
        :disabled="!isValid"
        class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
      >
        Add Event
      </button>
    </div>
  </form>
</template>
