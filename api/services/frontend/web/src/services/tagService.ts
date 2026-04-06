import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { Tag, NewTag, QueryResult } from '@/types'

const crud = createCRUDService<Tag, NewTag, Partial<Tag>, Record<string, never>>({
  basePath: '/api/v1/tags',
})

export const tagService = {
  ...crud,

  async getByTask(taskId: string): Promise<Tag[]> {
    const result = await request<QueryResult<Tag>>(`/api/v1/tasks/${taskId}/tags`)
    return result.items
  },

  async addToTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'DELETE' })
  },

  async getByContext(contextId: string): Promise<Tag[]> {
    const result = await request<QueryResult<Tag>>(`/api/v1/contexts/${contextId}/tags`)
    return result.items
  },

  async addToContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'DELETE' })
  },

  async getByNote(noteId: string): Promise<Tag[]> {
    const result = await request<QueryResult<Tag>>(`/api/v1/notes/${noteId}/tags`)
    return result.items
  },

  async addToNote(noteId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/notes/${noteId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromNote(noteId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/notes/${noteId}/tags/${tagId}`, { method: 'DELETE' })
  },
}
