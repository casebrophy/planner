import { describe, it, expect, beforeEach } from 'vitest'
import { scheduleService } from '@/services/scheduleService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('scheduleService', () => {
  describe('getSchedule', () => {
    it('calls request with correct params', async () => {
      const mockResponse = { items: [] }
      mockFetch.mockReturnValue(jsonResponse(mockResponse))

      const result = await scheduleService.getSchedule('2026-04-01T00:00:00Z', '2026-04-07T00:00:00Z')

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/schedule')
      expect(url).toContain('start=2026-04-01T00%3A00%3A00Z')
      expect(url).toContain('end=2026-04-07T00%3A00%3A00Z')
      expect(result).toEqual(mockResponse)
    })

    it('returns schedule items', async () => {
      const mockResponse = {
        items: [
          {
            type: 'time_block',
            id: 'block-1',
            title: 'Task 1',
            startsAt: '2026-04-02T09:00:00Z',
            endsAt: '2026-04-02T10:00:00Z',
            taskId: 'task-1',
            confirmed: false,
          },
          {
            type: 'event',
            id: 'event-1',
            title: 'Meeting',
            startsAt: '2026-04-02T14:00:00Z',
            endsAt: '2026-04-02T15:00:00Z',
            allDay: false,
          },
        ],
      }
      mockFetch.mockReturnValue(jsonResponse(mockResponse))

      const result = await scheduleService.getSchedule('2026-04-02T00:00:00Z', '2026-04-02T23:59:59Z')

      expect(result.items).toHaveLength(2)
      expect(result.items[0]?.type).toBe('time_block')
      expect(result.items[1]?.type).toBe('event')
    })
  })
})
