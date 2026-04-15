<script setup lang="ts">
import { ref, watch } from 'vue'
import type { ReceiptExtraction } from '@/services/receiptService'

interface Props {
  extraction: ReceiptExtraction
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'update', data: Partial<ReceiptExtraction>): void
  (e: 'confirm'): void
  (e: 'addSplits'): void
}>()

const localMerchant = ref(props.extraction.merchant)
const localDate = ref(props.extraction.date)

watch(() => props.extraction, (newVal) => {
  localMerchant.value = newVal.merchant
  localDate.value = newVal.date
}, { deep: true })

const handleMerchantChange = () => {
  emit('update', { merchant: localMerchant.value })
}

const handleDateChange = () => {
  emit('update', { date: localDate.value })
}

const formatCents = (cents: number) => {
  return `$${(cents / 100).toFixed(2)}`
}
</script>

<template>
  <div class="space-y-6">
    <!-- Merchant -->
    <div>
      <label class="block text-xs font-semibold text-gray-400 uppercase mb-2">Merchant</label>
      <input
        v-model="localMerchant"
        @change="handleMerchantChange"
        class="bg-gray-700 text-white rounded px-3 py-2 text-sm w-full border border-gray-600 focus:border-blue-500 focus:outline-none"
      />
    </div>

    <!-- Date -->
    <div>
      <label class="block text-xs font-semibold text-gray-400 uppercase mb-2">Date</label>
      <input
        v-model="localDate"
        type="date"
        @change="handleDateChange"
        class="bg-gray-700 text-white rounded px-3 py-2 text-sm w-full border border-gray-600 focus:border-blue-500 focus:outline-none"
      />
    </div>

    <!-- Totals Summary -->
    <div class="bg-gray-800 rounded-lg p-4 space-y-2 border border-gray-700">
      <div class="flex justify-between items-center">
        <span class="text-gray-400 text-sm">Subtotal</span>
        <span class="text-white font-medium">{{ formatCents(extraction.subtotal) }}</span>
      </div>
      <div class="flex justify-between items-center">
        <span class="text-gray-400 text-sm">Tax</span>
        <span class="text-white font-medium">{{ formatCents(extraction.tax) }}</span>
      </div>
      <div class="border-t border-gray-600 pt-2 flex justify-between items-center">
        <span class="text-white font-semibold">Total</span>
        <span class="text-blue-400 font-bold text-lg">{{ formatCents(extraction.total) }}</span>
      </div>
    </div>

    <!-- Line Items -->
    <div v-if="extraction.items && extraction.items.length > 0">
      <label class="block text-xs font-semibold text-gray-400 uppercase mb-3">Items</label>
      <div class="space-y-2">
        <div
          v-for="(item, idx) in extraction.items"
          :key="idx"
          class="bg-gray-800 rounded p-3 border border-gray-700 flex justify-between items-center"
        >
          <span class="text-white text-sm">{{ item.description }}</span>
          <span class="text-gray-400 text-sm">{{ formatCents(item.amount) }}</span>
        </div>
      </div>
    </div>

    <!-- Action Buttons -->
    <div class="flex gap-3 pt-4">
      <button
        @click="$emit('addSplits')"
        class="flex-1 px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white font-medium rounded-lg transition-colors text-sm"
      >
        Split with others
      </button>
      <button
        @click="$emit('confirm')"
        class="flex-1 px-4 py-2 bg-green-600 hover:bg-green-700 text-white font-medium rounded-lg transition-colors text-sm"
      >
        Looks good
      </button>
    </div>
  </div>
</template>
