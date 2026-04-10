import { createCRUDService } from './createCRUDService'
import { request } from './client'
import type { Transaction, UpdateTransaction, TransactionFilter, ImportResult, EnrichmentStatus } from '@/types'

const baseCrud = createCRUDService<Transaction, never, UpdateTransaction, TransactionFilter>({
  basePath: '/api/v1/transactions',
  mapFilter: (filter) => ({
    context_id: filter.contextId,
    source: filter.source,
    reviewed: filter.reviewed !== undefined ? String(filter.reviewed) : undefined,
    category: filter.category,
  }),
})

export const transactionService = {
  ...baseCrud,

  async importCSV(file: File, format?: string): Promise<ImportResult> {
    const formData = new FormData()
    formData.append('file', file)
    if (format) {
      formData.append('format', format)
    }

    const BASE_URL = import.meta.env.VITE_API_BASE_URL || ''
    const API_KEY = import.meta.env.VITE_API_KEY || ''

    const response = await fetch(`${BASE_URL}/api/v1/transactions/import`, {
      method: 'POST',
      headers: API_KEY ? { 'X-API-Key': API_KEY } : {},
      body: formData,
    })

    if (!response.ok) {
      const body = await response.json().catch(() => ({}))
      throw new Error((body as Record<string, string>).error || response.statusText)
    }

    return response.json() as Promise<ImportResult>
  },

  getEnrichmentStatus(): Promise<EnrichmentStatus> {
    return request<EnrichmentStatus>('/api/v1/transactions/enrichment-status')
  },
}
