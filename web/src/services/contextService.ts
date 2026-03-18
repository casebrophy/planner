import { request } from './client'
import type {
  Context,
  NewContext,
  UpdateContext,
  ContextFilter,
  ContextEvent,
  NewEvent,
  QueryResult,
  ListParams,
} from '@/types'

interface ContextListParams extends ListParams {
  filter?: ContextFilter
}

interface ContextQueryResponse {
  items: Context[]
  total: number
  page: number
  rowsPerPage: number
}

interface EventQueryResponse {
  items: ContextEvent[]
  total: number
  page: number
  rowsPerPage: number
}

export const contextService = {
  async list(params: ContextListParams = {}): Promise<QueryResult<Context>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
      orderBy: params.orderBy,
      status: params.filter?.status,
      title: params.filter?.title,
    }

    return request<ContextQueryResponse>('/api/v1/contexts', { params: queryParams })
  },

  async getById(id: string): Promise<Context> {
    return request<Context>(`/api/v1/contexts/${id}`)
  },

  async create(ctx: NewContext): Promise<Context> {
    return request<Context>('/api/v1/contexts', { method: 'POST', body: ctx })
  },

  async update(id: string, ctx: UpdateContext): Promise<Context> {
    return request<Context>(`/api/v1/contexts/${id}`, { method: 'PUT', body: ctx })
  },

  async delete(id: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${id}`, { method: 'DELETE' })
  },

  async listEvents(
    contextId: string,
    params: ListParams = {},
  ): Promise<QueryResult<ContextEvent>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
    }

    return request<EventQueryResponse>(`/api/v1/contexts/${contextId}/events`, {
      params: queryParams,
    })
  },

  async addEvent(contextId: string, event: NewEvent): Promise<ContextEvent> {
    return request<ContextEvent>(`/api/v1/contexts/${contextId}/events`, {
      method: 'POST',
      body: event,
    })
  },
}
