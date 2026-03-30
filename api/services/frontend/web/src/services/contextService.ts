import { request } from './client'
import { createCRUDService } from './createCRUDService'
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

const crud = createCRUDService<Context, NewContext, UpdateContext, ContextFilter>({
  basePath: '/api/v1/contexts',
  mapFilter: (f) => ({
    status: f.status,
    title: f.title,
  }),
})

export const contextService = {
  ...crud,

  async listEvents(
    contextId: string,
    params: ListParams = {},
  ): Promise<QueryResult<ContextEvent>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
    }
    return request<QueryResult<ContextEvent>>(`/api/v1/contexts/${contextId}/events`, {
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
