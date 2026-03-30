import { describe, it, expect, beforeEach } from 'vitest'
import { contextService } from '@/services/contextService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('contextService', () => {
  describe('list filter mapping', () => {
    it('maps ContextFilter fields to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 50 }))

      await contextService.list({
        filter: { status: 'active', title: 'Project' },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).toContain('title=Project')
    })
  })

  describe('listEvents', () => {
    it('fetches events for a context', async () => {
      const data = { items: [], total: 0, page: 1, rowsPerPage: 50 }
      mockFetch.mockReturnValue(jsonResponse(data))

      const result = await contextService.listEvents('ctx-1', { page: 1, rows: 50 })
      expect(result).toEqual(data)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts/ctx-1/events')
    })
  })

  describe('addEvent', () => {
    it('posts a new event', async () => {
      const event = { kind: 'note', content: 'test' }
      const created = { id: 'evt-1', contextId: 'ctx-1', ...event, createdAt: '2026-01-01' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await contextService.addEvent('ctx-1', event)
      expect(result).toEqual(created)

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/events')
      expect(options.method).toBe('POST')
    })
  })
})
