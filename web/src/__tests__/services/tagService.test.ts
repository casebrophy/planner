import { describe, it, expect, vi, beforeEach } from 'vitest'
import { tagService } from '@/services/tagService'

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function jsonResponse(data: unknown, status = 200) {
  return Promise.resolve({
    ok: status >= 200 && status < 300,
    status,
    statusText: 'OK',
    json: () => Promise.resolve(data),
  })
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('tagService', () => {
  describe('list', () => {
    it('fetches tags', async () => {
      const response = { items: [{ id: 't1', name: 'bug' }], total: 1, page: 1, rowsPerPage: 100 }
      mockFetch.mockReturnValue(jsonResponse(response))

      const result = await tagService.list()
      expect(result).toEqual(response)
    })
  })

  describe('create', () => {
    it('creates a tag', async () => {
      const tag = { id: 't1', name: 'feature' }
      mockFetch.mockReturnValue(jsonResponse(tag))

      const result = await tagService.create({ name: 'feature' })
      expect(result).toEqual(tag)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
    })
  })

  describe('delete', () => {
    it('deletes a tag', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await tagService.delete('t1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tags/t1')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('task tags', () => {
    it('gets tags by task', async () => {
      const tags = [{ id: 't1', name: 'bug' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByTask('task-1')
      expect(result).toEqual(tags)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tasks/task-1/tags')
    })

    it('adds tag to task', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await tagService.addToTask('task-1', 't1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/task-1/tags/t1')
      expect(options.method).toBe('POST')
    })

    it('removes tag from task', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await tagService.removeFromTask('task-1', 't1')

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('DELETE')
    })
  })

  describe('context tags', () => {
    it('gets tags by context', async () => {
      const tags = [{ id: 't1', name: 'important' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByContext('ctx-1')
      expect(result).toEqual(tags)
    })

    it('adds tag to context', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await tagService.addToContext('ctx-1', 't1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/tags/t1')
      expect(options.method).toBe('POST')
    })
  })
})
