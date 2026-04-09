import { describe, it, expect, beforeEach } from 'vitest'
import { activityLogService } from '@/services/activityLogService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('activityLogService', () => {
  describe('list filter mapping', () => {
    it('maps ActivityLogFilter fields to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 50 }))

      await activityLogService.list({
        page: 1,
        rows: 50,
        filter: {
          subjectType: 'task',
          subjectId: 'task-1',
          startDate: '2026-01-01',
          endDate: '2026-04-01',
        },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('subject_type=task')
      expect(url).toContain('subject_id=task-1')
      expect(url).toContain('start_date=2026-01-01')
      expect(url).toContain('end_date=2026-04-01')
    })

    it('omits undefined filter fields', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 50 }))

      await activityLogService.list({ filter: { subjectType: 'task' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('subject_type=task')
      expect(url).not.toContain('subject_id')
      expect(url).not.toContain('start_date')
      expect(url).not.toContain('end_date')
    })
  })

  describe('getBulkLogs', () => {
    it('calls bulk endpoint with correct query params', async () => {
      const mockResponse = {
        items: {
          'id-1': [{ id: 'log-1', subjectType: 'task', subjectId: 'id-1', loggedAt: '2026-04-01T00:00:00Z' }],
        },
      }
      mockFetch.mockReturnValue(jsonResponse(mockResponse))

      const result = await activityLogService.getBulkLogs(
        'task',
        ['id-1', 'id-2'],
        '2026-01-01T00:00:00Z',
        '2026-04-01T00:00:00Z',
      )

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/activity-logs/bulk')
      expect(url).toContain('subject_type=task')
      expect(url).toContain('subject_ids=id-1,id-2')
      expect(url).toContain('from=')
      expect(url).toContain('to=')
      expect(result.items).toEqual(mockResponse.items)
    })
  })

  describe('getStreaks', () => {
    it('calls the correct streaks endpoint', async () => {
      const streakData = { current: 5, longest: 12, totalCount: 42 }
      mockFetch.mockReturnValue(jsonResponse(streakData))

      const result = await activityLogService.getStreaks('task', 'task-1')

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/activity-logs/streaks/task/task-1')
      expect(result).toEqual(streakData)
    })

    it('includes subject type and id in the URL path', async () => {
      mockFetch.mockReturnValue(jsonResponse({ current: 0, longest: 0, totalCount: 0 }))

      await activityLogService.getStreaks('context', 'ctx-99')

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/activity-logs/streaks/context/ctx-99')
    })
  })
})
