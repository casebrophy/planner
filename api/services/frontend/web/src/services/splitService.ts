import { createCRUDService } from './createCRUDService'
import { request } from './client'
import type { Split, NewSplit, UpdateSplit } from '@/types'
import type { QueryResult } from '@/types'

const baseCrud = createCRUDService<Split, NewSplit, UpdateSplit>({
  basePath: '/api/v1/splits',
})

export const splitService = {
  ...baseCrud,

  async getByTransaction(transactionId: string): Promise<QueryResult<Split>> {
    return request<QueryResult<Split>>(`/api/v1/transactions/${transactionId}/splits`)
  },
}
