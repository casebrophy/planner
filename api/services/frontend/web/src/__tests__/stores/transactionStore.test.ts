import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTransactionStore } from '@/stores/transactionStore'
import type { Transaction, ImportResult } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/transactionService', () => ({
  transactionService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    importCSV: vi.fn(),
  },
}))

function makeTransaction(overrides: Partial<Transaction> = {}): Transaction {
  return {
    id: 'txn-1',
    source: 'chase',
    date: '2026-01-15',
    description: 'Coffee shop',
    amount: -4.5,
    reviewed: false,
    createdAt: '2026-01-15T10:00:00Z',
    ...overrides,
  }
}

describe('transactionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('unreviewedCount', () => {
    it('returns 0 when items list is empty', () => {
      const store = useTransactionStore()
      expect(store.unreviewedCount).toBe(0)
    })

    it('counts items where reviewed is false', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', reviewed: false }),
        makeTransaction({ id: 'txn-2', reviewed: true }),
        makeTransaction({ id: 'txn-3', reviewed: false }),
      ]
      expect(store.unreviewedCount).toBe(2)
    })

    it('counts items where reviewed is undefined (falsy)', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', reviewed: false }),
        makeTransaction({ id: 'txn-2', reviewed: true }),
        // reviewed omitted — will be undefined (falsy)
        { ...makeTransaction({ id: 'txn-3' }), reviewed: undefined as unknown as boolean },
      ]
      expect(store.unreviewedCount).toBe(2)
    })

    it('returns 0 when all items are reviewed', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', reviewed: true }),
        makeTransaction({ id: 'txn-2', reviewed: true }),
      ]
      expect(store.unreviewedCount).toBe(0)
    })
  })

  describe('totalSpend', () => {
    it('returns 0 when items list is empty', () => {
      const store = useTransactionStore()
      expect(store.totalSpend).toBe(0)
    })

    it('sums only negative amounts', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', amount: -4.5 }),
        makeTransaction({ id: 'txn-2', amount: -10.0 }),
        makeTransaction({ id: 'txn-3', amount: 100.0 }), // income/credit — excluded
      ]
      expect(store.totalSpend).toBeCloseTo(-14.5)
    })

    it('excludes zero-amount transactions', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', amount: 0 }),
        makeTransaction({ id: 'txn-2', amount: -20.0 }),
      ]
      expect(store.totalSpend).toBeCloseTo(-20.0)
    })

    it('returns 0 when all amounts are positive', () => {
      const store = useTransactionStore()
      store.items = [
        makeTransaction({ id: 'txn-1', amount: 50.0 }),
        makeTransaction({ id: 'txn-2', amount: 25.0 }),
      ]
      expect(store.totalSpend).toBe(0)
    })
  })

  describe('importCSV', () => {
    it('sets importing=true during the call and false after', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const importResult: ImportResult = { total: 5, imported: 5, skipped: 0 }

      let importingDuringCall = false
      vi.mocked(transactionService.importCSV).mockImplementation(async () => {
        // We cannot directly read the store here but we capture the sequence via timing
        importingDuringCall = true
        return importResult
      })
      vi.mocked(transactionService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 })

      const store = useTransactionStore()
      const promise = store.importCSV(new File(['data'], 'transactions.csv'))

      // importing should be true immediately after call starts
      expect(store.importing).toBe(true)
      await promise
      expect(importingDuringCall).toBe(true)
      expect(store.importing).toBe(false)
    })

    it('stores the result in lastImportResult', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const importResult: ImportResult = { total: 3, imported: 2, skipped: 1 }
      vi.mocked(transactionService.importCSV).mockResolvedValue(importResult)
      vi.mocked(transactionService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 })

      const store = useTransactionStore()
      const returned = await store.importCSV(new File(['data'], 'transactions.csv'))

      expect(store.lastImportResult).toEqual(importResult)
      expect(returned).toEqual(importResult)
    })

    it('calls transactionService.importCSV with file and format', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const importResult: ImportResult = { total: 1, imported: 1, skipped: 0 }
      vi.mocked(transactionService.importCSV).mockResolvedValue(importResult)
      vi.mocked(transactionService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 })

      const store = useTransactionStore()
      const file = new File(['csv'], 'bank.csv')
      await store.importCSV(file, 'chase')

      expect(transactionService.importCSV).toHaveBeenCalledWith(file, 'chase')
    })

    it('calls transactionService.importCSV without format when omitted', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const importResult: ImportResult = { total: 1, imported: 1, skipped: 0 }
      vi.mocked(transactionService.importCSV).mockResolvedValue(importResult)
      vi.mocked(transactionService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 })

      const store = useTransactionStore()
      const file = new File(['csv'], 'bank.csv')
      await store.importCSV(file)

      expect(transactionService.importCSV).toHaveBeenCalledWith(file, undefined)
    })

    it('refreshes the list after a successful import', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const importResult: ImportResult = { total: 2, imported: 2, skipped: 0 }
      vi.mocked(transactionService.importCSV).mockResolvedValue(importResult)
      vi.mocked(transactionService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 25 })

      const store = useTransactionStore()
      await store.importCSV(new File(['data'], 'transactions.csv'))

      // list is called with force=true to bypass the cache
      expect(transactionService.list).toHaveBeenCalled()
    })

    it('sets importing=false even when importCSV throws', async () => {
      const { transactionService } = await import('@/services/transactionService')
      vi.mocked(transactionService.importCSV).mockRejectedValue(new Error('network error'))

      const store = useTransactionStore()
      await expect(store.importCSV(new File(['data'], 'transactions.csv'))).rejects.toThrow('network error')
      expect(store.importing).toBe(false)
    })
  })

  describe('markReviewed', () => {
    it('calls crud.update with reviewed=true', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const original = makeTransaction({ id: 'txn-42', reviewed: false })
      const updated = { ...original, reviewed: true }
      vi.mocked(transactionService.update).mockResolvedValue(updated)

      const store = useTransactionStore()
      store.items = [original]

      await store.markReviewed('txn-42')

      expect(transactionService.update).toHaveBeenCalledWith('txn-42', { reviewed: true })
    })

    it('updates the item in the store after markReviewed', async () => {
      const { transactionService } = await import('@/services/transactionService')
      const original = makeTransaction({ id: 'txn-42', reviewed: false })
      const updated = { ...original, reviewed: true }
      vi.mocked(transactionService.update).mockResolvedValue(updated)

      const store = useTransactionStore()
      store.items = [original]

      await store.markReviewed('txn-42')

      expect(store.items[0]!.reviewed).toBe(true)
    })
  })
})
