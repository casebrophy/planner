<script setup lang="ts">
import type { Transaction } from '@/types'

const props = defineProps<{
  transaction: Transaction
}>()

const emit = defineEmits<{
  review: [id: string]
  click: [id: string]
}>()

function formatAmount(cents: number): string {
  const abs = Math.abs(cents)
  const dollars = (abs / 100).toFixed(2)
  return cents < 0 ? `-$${dollars}` : `$${dollars}`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  })
}
</script>

<template>
  <div
    class="flex items-center gap-4 px-4 py-3 border-b border-gray-800 hover:bg-gray-800/50 cursor-pointer transition-colors"
    :class="{ 'opacity-60': transaction.reviewed }"
    @click="emit('click', transaction.id)"
  >
    <span class="text-sm text-gray-400 w-16 shrink-0">
      {{ formatDate(transaction.date) }}
    </span>

    <div class="flex-1 min-w-0">
      <p class="text-sm text-gray-100 truncate">
        {{ transaction.cleanName || transaction.description }}
      </p>
      <p v-if="transaction.cleanName" class="text-xs text-gray-500 truncate">
        {{ transaction.description }}
      </p>
    </div>

    <span
      v-if="transaction.category"
      class="text-xs px-2 py-0.5 rounded-full bg-gray-700 text-gray-300 shrink-0"
    >
      {{ transaction.category }}
    </span>

    <span
      class="text-sm font-mono w-24 text-right shrink-0"
      :class="transaction.amount < 0 ? 'text-red-400' : 'text-green-400'"
    >
      {{ formatAmount(transaction.amount) }}
    </span>

    <button
      v-if="!transaction.reviewed"
      class="text-xs px-2 py-1 rounded bg-blue-600 hover:bg-blue-500 text-white shrink-0"
      @click.stop="emit('review', transaction.id)"
    >
      Review
    </button>
    <span v-else class="text-xs text-gray-500 w-14 text-center shrink-0">done</span>
  </div>
</template>
