import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { transactionService } from '@/services/transactionService'
import { createCRUDStore } from './createCRUDStore'
import type { Transaction, UpdateTransaction, TransactionFilter, ImportResult } from '@/types'

export const useTransactionStore = defineStore('transaction', () => {
  const crud = createCRUDStore<Transaction, never, UpdateTransaction, TransactionFilter>({
    name: 'transaction',
    service: transactionService,
    defaultOrderBy: 'date',
    defaultRowsPerPage: 25,
  })

  const importing = ref(false)
  const lastImportResult = ref<ImportResult | null>(null)

  const unreviewedCount = computed(() =>
    crud.items.value.filter((t) => !t.reviewed).length,
  )

  const totalSpend = computed(() =>
    crud.items.value
      .filter((t) => t.amount < 0)
      .reduce((sum, t) => sum + t.amount, 0),
  )

  async function importCSV(file: File, format?: string): Promise<ImportResult> {
    importing.value = true
    try {
      const result = await transactionService.importCSV(file, format)
      lastImportResult.value = result
      await crud.fetchList(true)
      return result
    } finally {
      importing.value = false
    }
  }

  async function markReviewed(id: string): Promise<void> {
    const reviewed = true
    await crud.update(id, { reviewed })
  }

  return {
    ...crud,
    importing,
    lastImportResult,
    unreviewedCount,
    totalSpend,
    importCSV,
    markReviewed,
  }
})
