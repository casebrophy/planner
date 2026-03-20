<script setup lang="ts">
import PageHeader from '@/components/layout/PageHeader.vue'
import ClarificationSession from '@/components/clarifications/ClarificationSession.vue'
import { useClarification } from '@/composables/useClarification'

const { pendingCount, refresh } = useClarification()
</script>

<template>
  <div>
    <PageHeader
      title="Clarifications"
      :subtitle="`${pendingCount} pending`"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <div class="p-6 max-w-2xl mx-auto">
      <ClarificationSession />
    </div>
  </div>
</template>
