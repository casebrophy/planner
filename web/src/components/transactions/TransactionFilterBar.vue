<script setup lang="ts">
import { computed } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'

const transactionStore = useTransactionStore()

const reviewedFilter = computed({
  get: () => transactionStore.filter.value.reviewed,
  set: (v: boolean | undefined) => {
    transactionStore.filter.value = { ...transactionStore.filter.value, reviewed: v }
    transactionStore.fetchList(true)
  },
})

const sourceFilter = computed({
  get: () => transactionStore.filter.value.source || '',
  set: (v: string) => {
    transactionStore.filter.value = {
      ...transactionStore.filter.value,
      source: v || undefined,
    }
    transactionStore.fetchList(true)
  },
})

function clearFilters() {
  transactionStore.filter.value = {} as any
  transactionStore.fetchList(true)
}

const sources = ['chase_checking', 'chase_credit', 'amex']
</script>

<template>
  <div class="flex items-center gap-3 text-sm">
    <div class="flex items-center gap-1.5">
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === undefined ? 'bg-gray-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = undefined"
      >
        All
      </button>
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === false ? 'bg-blue-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = false"
      >
        Needs Review
      </button>
      <button
        class="px-2 py-1 rounded transition-colors"
        :class="reviewedFilter === true ? 'bg-green-600 text-white' : 'text-gray-400 hover:text-white'"
        @click="reviewedFilter = true"
      >
        Reviewed
      </button>
    </div>

    <select
      :value="sourceFilter"
      class="bg-gray-700 text-gray-200 rounded px-2 py-1 border border-gray-600"
      @change="sourceFilter = ($event.target as HTMLSelectElement).value"
    >
      <option value="">All Sources</option>
      <option v-for="s in sources" :key="s" :value="s">{{ s }}</option>
    </select>

    <button
      class="text-gray-500 hover:text-gray-300 text-xs"
      @click="clearFilters"
    >
      Clear
    </button>
  </div>
</template>
