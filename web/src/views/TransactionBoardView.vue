<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'
import PageHeader from '@/components/layout/PageHeader.vue'
import TransactionImport from '@/components/transactions/TransactionImport.vue'
import TransactionFilterBar from '@/components/transactions/TransactionFilterBar.vue'
import TransactionRow from '@/components/transactions/TransactionRow.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import Pagination from '@/components/shared/Pagination.vue'

const store = useTransactionStore()

const totalPages = computed(() => Math.max(1, Math.ceil(store.total / store.rowsPerPage)))
const hasNext = computed(() => store.page < totalPages.value)
const hasPrev = computed(() => store.page > 1)

function formatSpend(cents: number): string {
  const abs = Math.abs(cents)
  return `$${(abs / 100).toFixed(2)}`
}

const subtitle = computed(() => {
  const parts = [`${store.total} transactions`]
  if (store.totalSpend !== 0) {
    parts.push(`${formatSpend(store.totalSpend)} spend`)
  }
  if (store.unreviewedCount > 0) {
    parts.push(`${store.unreviewedCount} unreviewed`)
  }
  return parts.join(' · ')
})

onMounted(() => {
  store.fetchList()
})
</script>

<template>
  <div>
    <PageHeader
      title="Transactions"
      :subtitle="subtitle"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="store.fetchList(true)"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <div class="p-6 space-y-4">
      <TransactionImport />

      <TransactionFilterBar />

      <LoadingSpinner v-if="store.loading && store.items.length === 0" />

      <EmptyState
        v-else-if="!store.loading && store.items.length === 0"
        title="No transactions found"
        message="Import a bank CSV to get started"
      />

      <div
        v-else
        class="bg-gray-900 rounded-lg border border-gray-800 overflow-hidden"
      >
        <TransactionRow
          v-for="transaction in store.items"
          :key="transaction.id"
          :transaction="transaction"
          @review="store.markReviewed"
        />
      </div>

      <Pagination
        v-if="totalPages > 1"
        :page="store.page"
        :total-pages="totalPages"
        :has-next="hasNext"
        :has-prev="hasPrev"
        @next="store.setPage(store.page + 1)"
        @prev="store.setPage(store.page - 1)"
      />
    </div>
  </div>
</template>
