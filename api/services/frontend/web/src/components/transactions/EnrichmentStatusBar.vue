<script setup lang="ts">
import type { EnrichmentStatus } from '@/types'

defineProps<{
  status: EnrichmentStatus
}>()

function isIdle(s: EnrichmentStatus): boolean {
  return s.active === 0 && s.pending === 0
}
</script>

<template>
  <div class="flex items-center gap-4 px-4 py-2.5 bg-gray-900 rounded-lg border border-gray-800 text-sm">
    <div class="flex items-center gap-1.5 text-gray-400">
      <div
        class="w-2 h-2 rounded-full"
        :class="isIdle(status) ? 'bg-gray-600' : 'bg-emerald-500 animate-pulse'"
      />
      <span class="font-medium">Enrichment</span>
    </div>

    <div
      v-if="!isIdle(status)"
      class="flex items-center gap-3 text-gray-500"
    >
      <span
        v-if="status.active > 0"
        class="text-emerald-400"
      >
        {{ status.active }} processing
      </span>
      <span
        v-if="status.pending > 0"
        class="text-amber-400"
      >
        {{ status.pending }} queued
      </span>
    </div>

    <div class="flex items-center gap-3 ml-auto text-gray-600">
      <span v-if="status.done > 0">{{ status.done }} done</span>
      <span
        v-if="status.failed > 0"
        class="text-red-400"
      >{{ status.failed }} failed</span>
      <span v-if="isIdle(status) && status.done === 0 && status.failed === 0">idle</span>
    </div>
  </div>
</template>
