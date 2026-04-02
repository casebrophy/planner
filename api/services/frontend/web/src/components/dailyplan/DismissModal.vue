<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  dismiss: [data: { reason: string; note?: string }]
  cancel: []
}>()

const selectedReason = ref('not_today')
const note = ref('')

const reasons = [
  { value: 'not_today', label: 'Not today' },
  { value: 'blocked', label: 'Blocked / waiting on someone' },
  { value: 'too_long', label: 'Takes too long' },
  { value: 'not_important', label: 'Not important right now' },
  { value: 'other', label: 'Other' },
]

function submit() {
  emit('dismiss', {
    reason: selectedReason.value,
    note: note.value || undefined,
  })
}
</script>

<template>
  <Teleport to="body">
    <Transition name="fade">
      <div
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
        @click.self="emit('cancel')"
      >
        <div class="relative bg-gray-900 border border-gray-800 rounded-xl p-6 w-full max-w-sm mx-4 shadow-xl">
          <h3 class="text-lg font-semibold text-gray-100 mb-4">Dismiss Task</h3>

          <div class="space-y-2 mb-4">
            <label
              v-for="r in reasons"
              :key="r.value"
              class="flex items-center gap-3 p-2 rounded-lg cursor-pointer hover:bg-gray-800/50 transition-colors"
              :class="{ 'bg-gray-800': selectedReason === r.value }"
            >
              <input
                v-model="selectedReason"
                type="radio"
                :value="r.value"
                class="accent-blue-500"
              />
              <span class="text-sm text-gray-300">{{ r.label }}</span>
            </label>
          </div>

          <textarea
            v-model="note"
            placeholder="Add a note (optional)"
            class="w-full p-2 rounded-lg bg-gray-800 border border-gray-700 text-sm text-gray-300 placeholder-gray-500 focus:border-blue-500 focus:outline-none resize-none"
            rows="2"
          />

          <div class="flex justify-end gap-2 mt-4">
            <button
              class="px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200 hover:bg-gray-800 rounded-lg transition-colors"
              @click="emit('cancel')"
            >
              Cancel
            </button>
            <button
              class="px-4 py-2 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
              @click="submit"
            >
              Dismiss
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
