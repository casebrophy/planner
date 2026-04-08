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
})
