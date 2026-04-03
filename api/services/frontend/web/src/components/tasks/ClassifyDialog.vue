<script setup lang="ts">
import { ref } from 'vue'
import { classifyService, type ClassifyResult } from '@/services/classifyService'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const running = ref(false)
const result = ref<ClassifyResult | null>(null)
const error = ref<string | null>(null)

async function classify() {
  running.value = true
  error.value = null
  try {
    result.value = await classifyService.classify()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Classification failed'
  } finally {
    running.value = false
  }
}

function close() {
  result.value = null
  error.value = null
  emit('close')
}
</script>

<template>
  <Teleport to="body">
    <Transition name="dialog">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center"
      >
        <div
          class="absolute inset-0 bg-black/50"
          @click="!running && close()"
        />
        <div class="relative bg-gray-900 border border-gray-800 rounded-xl shadow-xl max-w-md w-full mx-4 p-6">
          <!-- Confirm state -->
          <div v-if="!result && !error">
            <h3 class="text-lg font-semibold text-gray-100 mb-2">
              Classify Unlinked Tasks
            </h3>
            <p class="text-sm text-gray-400 mb-6">
              This will analyze tasks without a context assignment and suggest classifications.
              Low-confidence matches will create clarification cards for review.
            </p>
            <div class="flex justify-end gap-3">
              <button
                class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
                @click="close"
              >
                Cancel
              </button>
              <button
                :disabled="running"
                class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40"
                @click="classify"
              >
                {{ running ? 'Classifying...' : 'Classify' }}
              </button>
            </div>
          </div>

          <!-- Error state -->
          <div v-else-if="error">
            <h3 class="text-lg font-semibold text-red-400 mb-2">
              Classification Failed
            </h3>
            <p class="text-sm text-gray-400 mb-6">
              {{ error }}
            </p>
            <div class="flex justify-end">
              <button
                class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
                @click="close"
              >
                Close
              </button>
            </div>
          </div>

          <!-- Results state -->
          <div v-else-if="result">
            <h3 class="text-lg font-semibold text-gray-100 mb-4">
              Classification Complete
            </h3>
            <div class="space-y-3 mb-6">
              <div class="flex items-center justify-between bg-gray-800 rounded-lg px-4 py-3">
                <span class="text-sm text-gray-300">Tasks classified</span>
                <span class="text-lg font-semibold text-green-400">{{ result.classified }}</span>
              </div>
              <div class="flex items-center justify-between bg-gray-800 rounded-lg px-4 py-3">
                <span class="text-sm text-gray-300">Clarifications created</span>
                <span class="text-lg font-semibold text-amber-400">{{ result.clarificationsCreated }}</span>
              </div>
            </div>
            <div class="flex justify-end">
              <button
                class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
                @click="close"
              >
                Done
              </button>
            </div>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.dialog-enter-active,
.dialog-leave-active {
  transition: opacity 0.15s ease;
}
.dialog-enter-from,
.dialog-leave-to {
  opacity: 0;
}
</style>
