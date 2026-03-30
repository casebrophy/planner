import { describe, it, expect, beforeEach } from 'vitest'
import { taskService } from '@/services/taskService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('taskService', () => {
  describe('list filter mapping', () => {
    it('maps TaskFilter fields to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      await taskService.list({
        page: 1,
        rows: 20,
        filter: {
          status: 'todo',
          priority: 'high',
          contextId: 'ctx-1',
          startDueDate: '2026-01-01',
          endDueDate: '2026-12-31',
        },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=todo')
      expect(url).toContain('priority=high')
      expect(url).toContain('context_id=ctx-1')
      expect(url).toContain('start_due_date=2026-01-01')
      expect(url).toContain('end_due_date=2026-12-31')
    })

    it('omits undefined filter fields', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      await taskService.list({ filter: { status: 'todo' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=todo')
      expect(url).not.toContain('priority')
      expect(url).not.toContain('context_id')
    })
  })
})
