import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { RawInput } from '@/types/rawinput'
import type { QueryResult } from '@/types/query'

interface RawInputListParams {
  page?: number
  rows?: number
  status?: string
  sourceType?: string
  orderBy?: string
}

type NewRawInput = { sourceType: string; rawContent: string }
type UpdateRawInput = Record<string, never>
type RawInputFilter = { status?: string; sourceType?: string }

const crud = createCRUDService<RawInput, NewRawInput, UpdateRawInput, RawInputFilter>({
  basePath: '/api/v1/raw-inputs',
  mapFilter: (f) => ({
    status: f.status,
    source_type: f.sourceType,
  }),
})

export const rawinputService = {
  ...crud,

  async list(params: RawInputListParams = {}): Promise<QueryResult<RawInput>> {
    const queryParams: Record<string, string> = {}
    if (params.page) queryParams.page = String(params.page)
    if (params.rows) queryParams.rows = String(params.rows)
    if (params.status) queryParams.status = params.status
    if (params.sourceType) queryParams.source_type = params.sourceType
    if (params.orderBy) queryParams.orderBy = params.orderBy

    const res = await request<{ items: RawInput[]; total: number; page: number; rowsPerPage: number }>(
      '/api/v1/raw-inputs',
      { params: queryParams },
    )
    return { items: res.items, total: res.total, page: res.page, rowsPerPage: res.rowsPerPage }
  },

  async reprocess(id: string): Promise<RawInput> {
    return request<RawInput>(`/api/v1/raw-inputs/${id}/reprocess`, { method: 'POST' })
  },
}
