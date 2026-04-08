import { describe, it, expect, beforeEach } from 'vitest'
import { transactionService } from '@/services/transactionService'
import { setupMockFetch } from '../helpers/mockFetch'
import type { ImportResult } from '@/types'

const { mockFetch, jsonResponse, noContentResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('transactionService', () => {
  describe('list', () => {
    it('GETs /api/v1/transactions with pagination params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

      await transactionService.list({ page: 1, rows: 25, orderBy: 'date', filter: {} })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/transactions')
      expect(url).toContain('page=1')
      expect(url).toContain('rows=25')
    })

    it('maps filter fields to snake_case query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

      await transactionService.list({
        page: 1,
        rows: 25,
        orderBy: 'date',
        filter: { contextId: 'ctx-1', source: 'chase', reviewed: false, category: 'food' },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('context_id=ctx-1')
      expect(url).toContain('source=chase')
      expect(url).toContain('reviewed=false')
      expect(url).toContain('category=food')
    })

    it('omits undefined filter fields', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0 }))

      await transactionService.list({
        page: 1,
        rows: 25,
        orderBy: 'date',
        filter: { source: 'chase' },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).not.toContain('context_id')
      expect(url).not.toContain('reviewed')
      expect(url).not.toContain('category')
      expect(url).toContain('source=chase')
    })
  })

  describe('getById', () => {
    it('GETs /api/v1/transactions/:id', async () => {
      const txn = {
        id: 'txn-1',
        source: 'chase',
        date: '2026-01-01',
        description: 'Groceries',
        amount: -55.0,
        reviewed: false,
        createdAt: '2026-01-01T00:00:00Z',
      }
      mockFetch.mockReturnValue(jsonResponse(txn))

      const result = await transactionService.getById('txn-1')
      expect(result).toEqual(txn)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/transactions/txn-1')
    })
  })

  describe('update', () => {
    it('PATCHes /api/v1/transactions/:id with update body', async () => {
      const updated = {
        id: 'txn-2',
        source: 'chase',
        date: '2026-01-02',
        description: 'Coffee',
        amount: -4.5,
        reviewed: true,
        createdAt: '2026-01-02T00:00:00Z',
      }
      mockFetch.mockReturnValue(jsonResponse(updated))

      const result = await transactionService.update('txn-2', { reviewed: true })
      expect(result).toEqual(updated)

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/transactions/txn-2')
      expect(options.method).toBe('PATCH')
      expect(JSON.parse(options.body as string)).toEqual({ reviewed: true })
    })
  })

  describe('delete', () => {
    it('DELETEs /api/v1/transactions/:id', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await transactionService.delete('txn-3')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/transactions/txn-3')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('importCSV', () => {
    it('POSTs FormData to /api/v1/transactions/import', async () => {
      const importResult: ImportResult = { total: 10, imported: 8, skipped: 2 }
      mockFetch.mockReturnValue(jsonResponse(importResult))

      const file = new File(['date,description,amount\n2026-01-01,Coffee,-4.50'], 'bank.csv', {
        type: 'text/csv',
      })
      const result = await transactionService.importCSV(file)

      expect(result).toEqual(importResult)

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/transactions/import')
      expect(options.method).toBe('POST')
      expect(options.body).toBeInstanceOf(FormData)
    })

    it('includes the file in the FormData body', async () => {
      const importResult: ImportResult = { total: 3, imported: 3, skipped: 0 }
      mockFetch.mockReturnValue(jsonResponse(importResult))

      const file = new File(['csv content'], 'transactions.csv', { type: 'text/csv' })
      await transactionService.importCSV(file)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      const formData = options.body as FormData
      expect(formData.get('file')).toBe(file)
    })

    it('includes format in the FormData when provided', async () => {
      const importResult: ImportResult = { total: 5, imported: 5, skipped: 0 }
      mockFetch.mockReturnValue(jsonResponse(importResult))

      const file = new File(['csv content'], 'chase.csv', { type: 'text/csv' })
      await transactionService.importCSV(file, 'chase')

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      const formData = options.body as FormData
      expect(formData.get('format')).toBe('chase')
    })

    it('does not include format in FormData when not provided', async () => {
      const importResult: ImportResult = { total: 2, imported: 2, skipped: 0 }
      mockFetch.mockReturnValue(jsonResponse(importResult))

      const file = new File(['csv'], 'bank.csv', { type: 'text/csv' })
      await transactionService.importCSV(file)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      const formData = options.body as FormData
      expect(formData.get('format')).toBeNull()
    })

    it('throws an error when the response is not ok', async () => {
      mockFetch.mockReturnValue(
        jsonResponse({ error: 'unsupported format' }, 400),
      )

      const file = new File(['bad'], 'bad.csv', { type: 'text/csv' })
      await expect(transactionService.importCSV(file, 'unknown')).rejects.toThrow('unsupported format')
    })

    it('falls back to statusText when error body has no error field', async () => {
      mockFetch.mockReturnValue(
        jsonResponse({}, 400),
      )

      const file = new File(['bad'], 'bad.csv')
      await expect(transactionService.importCSV(file)).rejects.toThrow('Bad Request')
    })
  })
})
