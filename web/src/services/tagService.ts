import { request } from './client'
import type { Tag, NewTag, QueryResult, ListParams } from '@/types'

interface TagQueryResponse {
  items: Tag[]
  total: number
  page: number
  rowsPerPage: number
}

export const tagService = {
  async list(params: ListParams = {}): Promise<QueryResult<Tag>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
      orderBy: params.orderBy,
    }

    return request<TagQueryResponse>('/api/v1/tags', { params: queryParams })
  },

  async create(tag: NewTag): Promise<Tag> {
    return request<Tag>('/api/v1/tags', { method: 'POST', body: tag })
  },

  async delete(id: string): Promise<void> {
    return request<void>(`/api/v1/tags/${id}`, { method: 'DELETE' })
  },

  async getByTask(taskId: string): Promise<Tag[]> {
    return request<Tag[]>(`/api/v1/tasks/${taskId}/tags`)
  },

  async addToTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'DELETE' })
  },

  async getByContext(contextId: string): Promise<Tag[]> {
    return request<Tag[]>(`/api/v1/contexts/${contextId}/tags`)
  },

  async addToContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'DELETE' })
  },
}
