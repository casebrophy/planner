import { request } from './client'
import type { QueryResult, ListParams } from '@/types'

export interface CRUDServiceConfig<TFilter> {
  basePath: string
  mapFilter?: (filter: TFilter) => Record<string, string | number | undefined>
}

export interface CRUDService<T, TNew, TUpdate, TFilter> {
  list(params?: ListParams & { filter?: TFilter }): Promise<QueryResult<T>>
  getById(id: string): Promise<T>
  create(item: TNew): Promise<T>
  update(id: string, item: TUpdate): Promise<T>
  delete(id: string): Promise<void>
}

export function createCRUDService<T, TNew, TUpdate, TFilter = Record<string, never>>(
  config: CRUDServiceConfig<TFilter>,
): CRUDService<T, TNew, TUpdate, TFilter> {
  const { basePath, mapFilter } = config

  return {
    async list(params: ListParams & { filter?: TFilter } = {}): Promise<QueryResult<T>> {
      const queryParams: Record<string, string | number | undefined> = {
        page: params.page,
        rows: params.rows,
        orderBy: params.orderBy,
      }

      if (params.filter && mapFilter) {
        Object.assign(queryParams, mapFilter(params.filter))
      }

      return request<QueryResult<T>>(basePath, { params: queryParams })
    },

    async getById(id: string): Promise<T> {
      return request<T>(`${basePath}/${id}`)
    },

    async create(item: TNew): Promise<T> {
      return request<T>(basePath, { method: 'POST', body: item })
    },

    async update(id: string, item: TUpdate): Promise<T> {
      return request<T>(`${basePath}/${id}`, { method: 'PUT', body: item })
    },

    async delete(id: string): Promise<void> {
      return request<void>(`${basePath}/${id}`, { method: 'DELETE' })
    },
  }
}
