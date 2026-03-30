import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { Tag, NewTag } from '@/types'

const crud = createCRUDService<Tag, NewTag, Partial<Tag>, Record<string, never>>({
  basePath: '/api/v1/tags',
})

export const tagService = {
  ...crud,

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
