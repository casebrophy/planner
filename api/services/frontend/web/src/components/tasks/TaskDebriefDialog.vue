<script setup lang="ts">
import { ref } from 'vue'
import type { Task } from '@/types/task'
import { observationService } from '@/services/observationService'

const props = defineProps<{
  open: boolean
  taskId: string
  task: Task
}>()

const emit = defineEmits<{
  close: []
}>()

type Outcome = 'quick' | 'normal' | 'harder_than_expected' | 'blocked'

const selected = ref<Outcome | null>(null)
const note = ref('')
const submitting = ref(false)

const outcomes: { value: Outcome; label: string }[] = [
  { value: 'quick', label: '⚡ Quick' },
  { value: 'normal', label: '✓ Normal' },
  { value: 'harder_than_expected', label: '😓 Harder than expected' },
  { value: 'blocked', label: '🚫 Got blocked' },
]

async function handleSubmit() {
  if (!selected.value) return
  submitting.value = true
  try {
    await observationService.record({
      subjectType: 'task',
      subjectId: props.task.id,
      kind: 'debrief',
      data: {
        outcome: selected.value,
        note: note.value.trim() || undefined,
        contextId: props.task.contextId || null,
        priority: props.task.priority,
      },
    })
  } finally {
    submitting.value = false
    emit('close')
  }
}

async function handleSkip() {
  submitting.value = true
  try {
    await observationService.record({
      subjectType: 'task',
      subjectId: props.task.id,
      kind: 'debrief',
      data: {
        outcome: 'skipped',
        contextId: props.task.contextId || null,
        priority: props.task.priority,
      },
    })
  } finally {
    submitting.value = false
    emit('close')
  }
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex items-center justify-center bg-black/60"
    @click.self="handleSkip"
  >
    <div class="bg-gray-900 border border-gray-700 rounded-xl p-6 w-full max-w-md shadow-2xl">
      <h3 class="text-base font-semibold text-gray-100 mb-4">
        How did it go?
      </h3>
      <div class="grid grid-cols-2 gap-2 mb-4">
        <button
          v-for="o in outcomes"
          :key="o.value"
          :class="[
            'px-3 py-2 rounded-lg text-sm font-medium transition-colors text-left',
            selected === o.value
              ? 'bg-blue-600 text-white'
              : 'bg-gray-800 text-gray-300 hover:bg-gray-700',
          ]"
          @click="selected = o.value"
        >
          {{ o.label }}
        </button>
      </div>
      <textarea
        v-model="note"
        rows="2"
        placeholder="Optional: one sentence note (optional)"
        class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none mb-4"
      />
      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="px-4 py-2 text-sm font-medium text-gray-400 hover:text-gray-200 transition-colors"
          @click="handleSkip"
        >
          Skip
        </button>
        <button
          type="button"
          :disabled="!selected || submitting"
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="handleSubmit"
        >
          Submit
        </button>
      </div>
    </div>
  </div>
</template>
