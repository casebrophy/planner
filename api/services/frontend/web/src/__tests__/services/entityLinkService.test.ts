import { describe, it, expect, beforeEach } from 'vitest'
import { entityLinkService } from '@/services/entityLinkService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse, noContentResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('entityLinkService', () => {
  describe('listByEntity', () => {
    it('passes correct query params and returns items', async () => {
      const items = [
        {
          id: 'link-1',
          sourceType: 'note',
          sourceId: 'abc-123',
          targetType: 'task',
          targetId: 'def-456',
          confidence: 0.9,
          kind: 'ai_suggested',
          createdAt: '2026-01-01T00:00:00Z',
        },
      ]
      mockFetch.mockReturnValue(jsonResponse({ items, total: 1 }))

      const result = await entityLinkService.listByEntity('note', 'abc-123')
      expect(result).toEqual(items)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/entity-links')
      expect(url).toContain('entity_type=note')
      expect(url).toContain('entity_id=abc-123')
    })
  })

  describe('create', () => {
    it('posts to correct endpoint with body', async () => {
      const link = {
        sourceType: 'note' as const,
        sourceId: 'a',
        targetType: 'event' as const,
        targetId: 'b',
      }
      const created = { id: 'x', ...link, confidence: 1, kind: 'manual', createdAt: '' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await entityLinkService.create(link)
      expect(result).toEqual(created)

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/entity-links')
      expect(options.method).toBe('POST')
      expect(JSON.parse(options.body as string)).toEqual(link)
    })
  })

  describe('delete', () => {
    it('calls correct URL with DELETE method', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await entityLinkService.delete('link-id-123')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/entity-links/link-id-123')
      expect(options.method).toBe('DELETE')
    })
  })
})
