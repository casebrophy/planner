<script setup lang="ts">
import { ref, computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { ClarificationKind, ClarificationKindLabels, ClarificationKindColors } from '@/types/enums'
import type { ClarificationItem } from '@/types'

const props = defineProps<{
  item: ClarificationItem
}>()

const emit = defineEmits<{
  resolve: [answer: Record<string, unknown>]
  snooze: [hours: number]
  dismiss: []
}>()

const debriefAnswer = ref('')

const kindLabel = computed(() => ClarificationKindLabels[props.item.kind] ?? props.item.kind)
const kindColor = computed(() => ClarificationKindColors[props.item.kind] ?? '#6b7280')
const age = computed(() => formatDistanceToNow(new Date(props.item.createdAt), { addSuffix: true }))

const options = computed(() => {
  if (!props.item.answerOptions) return {}
  return typeof props.item.answerOptions === 'string'
    ? JSON.parse(props.item.answerOptions as unknown as string)
    : props.item.answerOptions
})

function resolveWithValue(answer: Record<string, unknown>) {
  emit('resolve', answer)
}

function resolveDebrief() {
  if (debriefAnswer.value.trim()) {
    emit('resolve', { response: debriefAnswer.value.trim() })
  }
}
</script>

<template>
  <div
    class="bg-gray-800 rounded-xl p-6 border-l-4"
    :style="{ borderLeftColor: kindColor }"
  >
    <!-- Header -->
    <div class="flex items-center gap-2 mb-3">
      <span
        class="px-2 py-0.5 rounded text-xs font-medium"
        :style="{ backgroundColor: kindColor + '22', color: kindColor }"
      >
        {{ kindLabel }}
      </span>
      <span class="text-gray-500 text-xs">{{ age }}</span>
    </div>

    <!-- Question -->
    <h3 class="text-lg font-semibold text-gray-100 mb-2">
      {{ item.question }}
    </h3>

    <!-- Reasoning (if present) -->
    <p
      v-if="item.reasoning"
      class="text-sm text-gray-400 mb-4"
    >
      {{ item.reasoning }}
    </p>

    <!-- Kind-specific actions -->
    <div class="mt-4">
      <!-- Context Assignment -->
      <div
        v-if="item.kind === ClarificationKind.ContextAssignment"
        class="flex flex-col gap-2"
      >
        <button
          v-if="options.suggested_context_id"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ context_id: options.suggested_context_id })"
        >
          Confirm suggested context
        </button>
        <button
          v-for="alt in (options.alternatives ?? [])"
          :key="alt"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
          @click="resolveWithValue({ context_id: alt })"
        >
          {{ alt }}
        </button>
      </div>

      <!-- Inactivity Prompt / Stale Task -->
      <div
        v-else-if="item.kind === ClarificationKind.InactivityPrompt || item.kind === ClarificationKind.StaleTask"
        class="flex gap-2"
      >
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'extend' })"
        >
          Still active
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-amber-600 hover:bg-amber-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'note' })"
        >
          Add note
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-red-600 hover:bg-red-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'close' })"
        >
          Close
        </button>
      </div>

      <!-- Ambiguous Action -->
      <div
        v-else-if="item.kind === ClarificationKind.AmbiguousAction"
        class="flex flex-col gap-2"
      >
        <button
          v-for="(interp, idx) in (options.interpretations ?? [])"
          :key="idx"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors text-left"
          @click="resolveWithValue({ selected: idx })"
        >
          {{ typeof interp === 'string' ? interp : JSON.stringify(interp) }}
        </button>
      </div>

      <!-- Ambiguous Deadline -->
      <div
        v-else-if="item.kind === ClarificationKind.AmbiguousDeadline"
        class="flex flex-col gap-2"
      >
        <input
          type="datetime-local"
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @change="(e) => resolveWithValue({ due_date: new Date((e.target as HTMLInputElement).value).toISOString() })"
        >
      </div>

      <!-- New Context -->
      <div
        v-else-if="item.kind === ClarificationKind.NewContext"
        class="flex gap-2"
      >
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'confirm' })"
        >
          Confirm
        </button>
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-500 rounded-lg transition-colors"
          @click="resolveWithValue({ action: 'merge' })"
        >
          Merge
        </button>
      </div>

      <!-- Context Debrief -->
      <div
        v-else-if="item.kind === ClarificationKind.ContextDebrief"
        class="flex flex-col gap-2"
      >
        <textarea
          v-model="debriefAnswer"
          rows="3"
          placeholder="Your answer..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
        />
        <button
          :disabled="!debriefAnswer.trim()"
          class="w-full px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          @click="resolveDebrief"
        >
          Submit
        </button>
      </div>

      <!-- Voice Reference -->
      <div
        v-else-if="item.kind === ClarificationKind.VoiceReference"
        class="flex flex-col gap-2"
      >
        <input
          type="text"
          placeholder="Corrected reference..."
          class="w-full bg-gray-700 border border-gray-600 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
          @keyup.enter="(e) => resolveWithValue({ resolved_text: (e.target as HTMLInputElement).value })"
        >
      </div>

      <!-- Fallback -->
      <div
        v-else
        class="flex gap-2"
      >
        <button
          class="flex-1 px-4 py-2.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-500 rounded-lg transition-colors"
          @click="resolveWithValue({ acknowledged: true })"
        >
          Acknowledge
        </button>
      </div>
    </div>

    <!-- Snooze / Dismiss -->
    <div class="flex gap-2 mt-3">
      <button
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
        @click="emit('snooze', 24)"
      >
        Snooze 24h
      </button>
      <button
        class="flex-1 px-3 py-2 text-sm text-gray-400 bg-transparent border border-gray-700 hover:border-gray-600 rounded-lg transition-colors"
        @click="emit('dismiss')"
      >
        Dismiss
      </button>
    </div>
  </div>
</template>
