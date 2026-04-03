<script setup lang="ts">
import { ref } from 'vue'
import { useActivityLogStore } from '@/stores/activityLogStore'

const props = defineProps<{
  subjectType: string
  subjectId: string
}>()

const store = useActivityLogStore()
const showInput = ref(false)
const value = ref('')
const logging = ref(false)

async function log() {
  logging.value = true
  try {
    await store.logActivity(props.subjectType, props.subjectId, value.value || undefined)
    showInput.value = false
    value.value = ''
  } finally {
    logging.value = false
  }
}
</script>

<template>
  <div>
    <button
      v-if="!showInput"
      class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors inline-flex items-center gap-1.5"
      @click="showInput = true"
    >
      <svg
        class="w-4 h-4"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M12 4v16m8-8H4"
        />
      </svg>
      Log
    </button>
    <div
      v-else
      class="flex gap-2"
    >
      <input
        v-model="value"
        type="text"
        class="flex-1 bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-1.5 text-sm focus:outline-none focus:border-blue-500"
        placeholder="Value (optional)"
        @keyup.enter="log"
      >
      <button
        :disabled="logging"
        class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40"
        @click="log"
      >
        {{ logging ? '...' : 'Save' }}
      </button>
      <button
        class="px-3 py-1.5 text-sm text-gray-400 hover:text-gray-100 transition-colors"
        @click="showInput = false; value = ''"
      >
        Cancel
      </button>
    </div>
  </div>
</template>
