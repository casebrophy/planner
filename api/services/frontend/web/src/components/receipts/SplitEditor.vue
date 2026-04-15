<script setup lang="ts">
import { ref, computed } from 'vue'
import type { NewSplit } from '@/types'

const props = defineProps<{
  totalCents: number
  transactionId: string
}>()

const emit = defineEmits<{
  done: [splits: NewSplit[]]
}>()

interface SplitEntry {
  partyName: string
  amountDollars: string
  venmoHandle: string
}

const splits = ref<SplitEntry[]>([])

const totalDollars = computed(() => (props.totalCents / 100).toFixed(2))

const splitTotal = computed(() => {
  return splits.value.reduce((sum, split) => {
    const amount = parseFloat(split.amountDollars) || 0
    return sum + amount
  }, 0)
})

const remaining = computed(() => {
  const total = parseFloat(totalDollars.value)
  return (total - splitTotal.value).toFixed(2)
})

function addParty() {
  splits.value.push({
    partyName: '',
    amountDollars: '',
    venmoHandle: '',
  })
}

function removeParty(index: number) {
  splits.value.splice(index, 1)
}

function splitEvenly() {
  if (splits.value.length === 0) return

  const numPeople = splits.value.length + 1 // +1 for "you"
  const amountPerPerson = (parseFloat(totalDollars.value) / numPeople).toFixed(2)

  splits.value.forEach(split => {
    split.amountDollars = amountPerPerson
  })
}

function getVenmoLink(split: SplitEntry): string {
  if (!split.venmoHandle) return ''
  const amount = (parseFloat(split.amountDollars) || 0).toFixed(2)
  return `venmo://paycharge?txn=charge&recipients=${encodeURIComponent(split.venmoHandle)}&amount=${amount}&note=`
}

function confirmSplits() {
  const newSplits: NewSplit[] = splits.value.map(split => ({
    transactionId: props.transactionId,
    partyName: split.partyName,
    amount: Math.round(parseFloat(split.amountDollars || '0') * 100),
    venmoHandle: split.venmoHandle || undefined,
  }))
  emit('done', newSplits)
}
</script>

<template>
  <div class="space-y-4">
    <!-- Remaining amount -->
    <div class="bg-gray-800 border border-gray-700 rounded-lg p-4">
      <div class="text-sm text-gray-400">Total</div>
      <div class="text-2xl font-semibold text-white">${{ totalDollars }}</div>
      <div class="mt-2 text-sm text-gray-400">Remaining: <span class="text-white font-semibold">${{ remaining }}</span></div>
    </div>

    <!-- Splits list -->
    <div class="space-y-3">
      <div v-for="(split, index) in splits" :key="index" class="bg-gray-800 border border-gray-700 rounded-lg p-4 space-y-3">
        <!-- Name -->
        <input
          v-model="split.partyName"
          type="text"
          placeholder="Party name"
          class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none"
        />

        <!-- Amount -->
        <div class="flex gap-2 items-end">
          <div class="flex-1">
            <label class="block text-xs text-gray-400 mb-1">Amount ($)</label>
            <input
              v-model="split.amountDollars"
              type="number"
              placeholder="0.00"
              step="0.01"
              class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none"
            />
          </div>
          <button
            @click="removeParty(index)"
            class="px-3 py-2 bg-red-900 hover:bg-red-800 text-white rounded text-sm"
          >
            Remove
          </button>
        </div>

        <!-- Venmo handle -->
        <input
          v-model="split.venmoHandle"
          type="text"
          placeholder="Venmo handle (optional)"
          class="w-full px-3 py-2 bg-gray-700 border border-gray-600 rounded text-white placeholder-gray-500 focus:border-blue-500 focus:outline-none"
        />

        <!-- Venmo link button -->
        <div v-if="split.venmoHandle" class="pt-2">
          <a
            :href="getVenmoLink(split)"
            class="inline-block px-4 py-2 bg-blue-600 hover:bg-blue-700 text-white text-sm rounded"
          >
            Send via Venmo
          </a>
        </div>
      </div>
    </div>

    <!-- Action buttons -->
    <div class="flex gap-2">
      <button
        @click="addParty"
        class="flex-1 px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white border border-gray-700 rounded-lg"
      >
        Add Party
      </button>
      <button
        v-if="splits.length > 0"
        @click="splitEvenly"
        class="flex-1 px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white border border-gray-700 rounded-lg"
      >
        Split Evenly
      </button>
    </div>

    <!-- Confirm button -->
    <button
      @click="confirmSplits"
      :disabled="splits.length === 0 || splits.some(s => !s.partyName || !s.amountDollars)"
      class="w-full px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-gray-700 disabled:text-gray-500 text-white rounded-lg font-semibold"
    >
      Confirm Splits
    </button>
  </div>
</template>
