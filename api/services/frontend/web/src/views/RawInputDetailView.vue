<script setup lang="ts">
import { watchEffect, ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useRawInputStore } from '@/stores/rawinputStore'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import type { StepResult } from '@/types/rawinput'

const route = useRoute()
const store = useRawInputStore()
const rawContentExpanded = ref(false)

const id = computed(() => route.params.id as string)

watchEffect(async () => {
  if (id.value) {
    await store.fetchById(id.value)
  }
})

const item = computed(() => store.selectedItem)

const pipelineSteps = computed(() => {
  if (!item.value?.result) return []
  const r = item.value.result
  const steps: { name: string; label: string; result?: StepResult }[] = [
    { name: 'sanitize', label: 'Sanitize', result: r.sanitize },
    { name: 'extraction', label: 'AI Extraction', result: r.extraction },
    { name: 'contextMatch', label: 'Context Match', result: r.contextMatch },
    { name: 'tasks', label: 'Task Creation', result: r.tasks },
    { name: 'events', label: 'Event Creation', result: r.events },
    { name: 'notes', label: 'Note Creation', result: r.notes },
  ]
  return steps
})

function statusIcon(status?: string): string {
  if (status === 'completed') return '\u2713'
  if (status === 'failed') return '\u2717'
  if (status === 'skipped') return '\u2014'
  return '\u00B7'
}

function statusColor(status?: string): string {
  if (status === 'completed') return 'text-green-600 dark:text-green-400'
  if (status === 'failed') return 'text-red-600 dark:text-red-400'
  return 'text-gray-400 dark:text-gray-500'
}

function formatDetail(detail?: Record<string, unknown>): string {
  if (!detail) return ''
  return Object.entries(detail)
    .map(([k, v]) => {
      if (Array.isArray(v)) return `${k}: ${v.length}`
      return `${k}: ${v}`
    })
    .join(' \u00B7 ')
}

function formatDate(iso?: string): string {
  if (!iso) return '\u2014'
  return new Date(iso).toLocaleString()
}

function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    processing: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    processed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  }
  return map[status] ?? 'bg-gray-100 text-gray-800'
}

async function handleReprocess() {
  if (!item.value) return
  await store.reprocess(item.value.id)
  await store.fetchById(item.value.id)
}
</script>

<template>
  <div class="p-6 space-y-6">
    <LoadingSpinner v-if="store.loading && !item" />

    <template v-else-if="item">
      <!-- Header -->
      <div class="space-y-2">
        <div class="flex items-center gap-3">
          <span class="font-mono text-sm text-gray-500 dark:text-gray-400">
            {{ item.sourceType }}
          </span>
          <span
            :class="[
              'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
              statusBadgeClass(item.status),
            ]"
          >
            {{ item.status }}
          </span>
        </div>
        <div class="text-xs text-gray-500 dark:text-gray-400 space-y-0.5">
          <div>Created: {{ formatDate(item.createdAt) }}</div>
          <div v-if="item.processedAt">
            Processed: {{ formatDate(item.processedAt) }}
          </div>
          <div v-if="item.retryCount > 0">
            Retries: {{ item.retryCount }} / {{ item.maxRetries }}
          </div>
        </div>
      </div>

      <!-- Error -->
      <div
        v-if="item.error"
        class="p-3 rounded-lg bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800"
      >
        <p class="text-sm font-medium text-red-800 dark:text-red-200">
          Error
        </p>
        <p class="text-sm text-red-600 dark:text-red-300 mt-1 break-words">
          {{ item.error }}
        </p>
      </div>

      <!-- Pipeline Steps -->
      <div v-if="pipelineSteps.length > 0">
        <h3 class="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
          Pipeline Steps
        </h3>
        <div class="space-y-1">
          <div
            v-for="step in pipelineSteps"
            :key="step.name"
            class="flex items-start gap-2 py-1.5 px-2 rounded text-sm"
          >
            <span :class="['font-mono text-base leading-5 w-5 text-center shrink-0', statusColor(step.result?.status)]">
              {{ statusIcon(step.result?.status) }}
            </span>
            <div class="min-w-0">
              <span class="text-gray-800 dark:text-gray-200">{{ step.label }}</span>
              <span
                v-if="step.result?.detail"
                class="ml-2 text-xs text-gray-500 dark:text-gray-400"
              >
                {{ formatDetail(step.result.detail) }}
              </span>
              <p
                v-if="step.result?.status === 'failed' && step.result.detail?.error"
                class="text-xs text-red-500 mt-0.5 break-words"
              >
                {{ step.result.detail.error }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <div v-else-if="item.status === 'processed' || item.status === 'failed'">
        <p class="text-sm text-gray-400 dark:text-gray-500 italic">
          No pipeline result recorded (processed before tracking was added).
        </p>
      </div>

      <!-- Raw Content -->
      <div>
        <button
          class="text-sm font-medium text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100 flex items-center gap-1"
          @click="rawContentExpanded = !rawContentExpanded"
        >
          <span class="text-xs">{{ rawContentExpanded ? '\u25BC' : '\u25B6' }}</span>
          Raw Content
        </button>
        <pre
          v-if="rawContentExpanded"
          class="mt-2 p-3 rounded-lg bg-gray-50 dark:bg-gray-800 text-xs text-gray-700 dark:text-gray-300 overflow-x-auto whitespace-pre-wrap break-words max-h-64 overflow-y-auto"
        >{{ item.rawContent }}</pre>
      </div>

      <!-- Actions -->
      <div class="pt-2 border-t border-gray-200 dark:border-gray-700">
        <button
          v-if="item.status === 'failed' || item.status === 'processed'"
          class="text-sm px-3 py-1.5 rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
          :disabled="store.loading"
          @click="handleReprocess"
        >
          Reprocess
        </button>
      </div>
    </template>

    <div
      v-else
      class="text-center py-12 text-gray-400"
    >
      Raw input not found.
    </div>
  </div>
</template>
