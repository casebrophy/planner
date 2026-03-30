import { request } from './client'
import type { ClarificationItem, ClarificationCountResponse } from '@/types'
import type { QueryResult, ListParams } from '@/types/query'

interface ClarificationListParams extends ListParams {
  status?: string
}

interface ClarificationQueryResponse {
  items: ClarificationItem[]
  total: number
  page: number
  rowsPerPage: number
}

export const clarificationService = {
  async queryQueue(
    params: ClarificationListParams = {},
  ): Promise<QueryResult<ClarificationItem>> {
    const queryParams: Record<string, string> = {}
    if (params.page) queryParams.page = String(params.page)
    if (params.rows) queryParams.rows_per_page = String(params.rows)
    if (params.status) queryParams.status = params.status
    if (params.orderBy) queryParams.orderBy = params.orderBy

    const res = await request<ClarificationQueryResponse>('/api/v1/clarifications', {
      params: queryParams,
    })
    return { items: res.items, total: res.total, page: res.page, rowsPerPage: res.rowsPerPage }
  },

  async queryByID(id: string): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}`)
  },

  async countPending(): Promise<number> {
    const res = await request<ClarificationCountResponse>('/api/v1/clarifications/count')
    return res.count
  },

  async resolve(id: string, answer: Record<string, unknown>): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/resolve`, {
      method: 'POST',
      body: { answer },
    })
  },

  async snooze(id: string, hours: number = 24): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/snooze`, {
      method: 'POST',
      body: { hours },
    })
  },

  async dismiss(id: string): Promise<ClarificationItem> {
    return request<ClarificationItem>(`/api/v1/clarifications/${id}/dismiss`, {
      method: 'POST',
    })
  },
}
