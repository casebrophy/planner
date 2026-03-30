import { describe, it, expect, beforeEach } from 'vitest'
import { tagService } from '@/services/tagService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse, noContentResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('tagService', () => {
  describe('getByTask', () => {
    it('fetches tags for a task', async () => {
      const tags = [{ id: 't1', name: 'urgent' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByTask('task-1')
      expect(result).toEqual(tags)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tasks/task-1/tags')
    })
  })

  describe('addToTask', () => {
    it('posts tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.addToTask('task-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/task-1/tags/tag-1')
      expect(options.method).toBe('POST')
    })
  })

  describe('removeFromTask', () => {
    it('deletes tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.removeFromTask('task-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/task-1/tags/tag-1')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('getByContext', () => {
    it('fetches tags for a context', async () => {
      const tags = [{ id: 't1', name: 'work' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByContext('ctx-1')
      expect(result).toEqual(tags)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts/ctx-1/tags')
    })
  })

  describe('addToContext', () => {
    it('posts tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.addToContext('ctx-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/tags/tag-1')
      expect(options.method).toBe('POST')
    })
  })

  describe('removeFromContext', () => {
    it('deletes tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.removeFromContext('ctx-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/tags/tag-1')
      expect(options.method).toBe('DELETE')
    })
  })
})
