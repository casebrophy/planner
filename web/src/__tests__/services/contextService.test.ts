import { describe, it, expect, vi, beforeEach } from 'vitest'
import { contextService } from '@/services/contextService'

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

describe('contextService', () => {
  describe('list', () => {
    it('fetches contexts with default params', async () => {
      const response = { items: [], total: 0, page: 1, rowsPerPage: 50 }
      mockFetch.mockReturnValue(jsonResponse(response))

      const result = await contextService.list()
      expect(result).toEqual(response)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts')
    })

    it('passes filter params', async () => {
      const response = { items: [], total: 0, page: 1, rowsPerPage: 50 }
      mockFetch.mockReturnValue(jsonResponse(response))

      await contextService.list({ filter: { status: 'active', title: 'test' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).toContain('title=test')
    })
  })

  describe('getById', () => {
    it('fetches a single context', async () => {
      const ctx = { id: 'ctx-1', title: 'Project', status: 'active' }
      mockFetch.mockReturnValue(jsonResponse(ctx))

      const result = await contextService.getById('ctx-1')
      expect(result).toEqual(ctx)
    })
  })

  describe('create', () => {
    it('posts a new context', async () => {
      const newCtx = { title: 'New Project', description: 'Desc' }
      const created = { ...newCtx, id: 'new-id', status: 'active' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await contextService.create(newCtx)
      expect(result).toEqual(created)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
    })
  })

  describe('update', () => {
    it('puts context updates', async () => {
      const update = { title: 'Updated' }
      const updated = { id: 'ctx-1', title: 'Updated', status: 'active' }
      mockFetch.mockReturnValue(jsonResponse(updated))

      const result = await contextService.update('ctx-1', update)
      expect(result).toEqual(updated)
    })
  })

  describe('delete', () => {
    it('deletes a context', async () => {
      mockFetch.mockReturnValue(
        Promise.resolve({ ok: true, status: 204, json: () => Promise.resolve(undefined) }),
      )

      await contextService.delete('ctx-1')

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('DELETE')
    })
  })

  describe('events', () => {
    it('lists events for a context', async () => {
      const response = { items: [], total: 0, page: 1, rowsPerPage: 50 }
      mockFetch.mockReturnValue(jsonResponse(response))

      const result = await contextService.listEvents('ctx-1')
      expect(result).toEqual(response)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts/ctx-1/events')
    })

    it('adds an event', async () => {
      const event = { kind: 'note', content: 'Hello' }
      const created = { ...event, id: 'evt-1', contextId: 'ctx-1', createdAt: '2025-01-01T00:00:00Z' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await contextService.addEvent('ctx-1', event)
      expect(result).toEqual(created)

      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
    })
  })
})
