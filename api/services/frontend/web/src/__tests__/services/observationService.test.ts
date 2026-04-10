import { describe, it, expect, beforeEach } from 'vitest'
import { observationService } from '@/services/observationService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('observationService', () => {
  describe('queryBySubject', () => {
    it('fetches observations for a specific subject', async () => {
      const mockData = {
        items: [
          {
            id: 'obs-1',
            subjectType: 'task',
            subjectId: 'task-1',
            kind: 'debrief',
            data: { outcome: 'completed' },
            source: 'user',
            confidence: 1,
            weight: 1,
            createdAt: '2026-04-09T12:00:00Z',
          },
        ],
        total: 1,
        page: 1,
        rowsPerPage: 20,
      }

      mockFetch.mockReturnValue(jsonResponse(mockData))

      const result = await observationService.queryBySubject('task', 'task-1')

      expect(result).toEqual(mockData.items)
      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/observations/task/task-1')
    })
  })

  describe('queryByKind', () => {
    it('fetches observations by kind with query params', async () => {
      const mockData = {
        items: [
          {
            id: 'obs-1',
            subjectType: 'task',
            subjectId: 'task-1',
            kind: 'debrief',
            data: { outcome: 'skipped', contextId: 'ctx-1', priority: 'high', energy: 'high' },
            source: 'user',
            confidence: 1,
            weight: 1,
            createdAt: '2026-04-09T12:00:00Z',
          },
          {
            id: 'obs-2',
            subjectType: 'task',
            subjectId: 'task-2',
            kind: 'debrief',
            data: { outcome: 'completed', contextId: 'ctx-1', priority: 'high', energy: 'high' },
            source: 'user',
            confidence: 1,
            weight: 1,
            createdAt: '2026-04-09T13:00:00Z',
          },
        ],
        total: 2,
        page: 1,
        rowsPerPage: 100,
      }

      mockFetch.mockReturnValue(jsonResponse(mockData))

      const result = await observationService.queryByKind('task', 'debrief')

      expect(result).toEqual(mockData.items)
      expect(result).toHaveLength(2)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/observations')
      expect(url).toContain('subject_type=task')
      expect(url).toContain('kind=debrief')
      expect(url).toContain('rows_per_page=100')
    })

    it('returns empty array when no observations found', async () => {
      const mockData = {
        items: [],
        total: 0,
        page: 1,
        rowsPerPage: 100,
      }

      mockFetch.mockReturnValue(jsonResponse(mockData))

      const result = await observationService.queryByKind('task', 'debrief')

      expect(result).toEqual([])
    })
  })

  describe('record', () => {
    it('posts an observation with all fields', async () => {
      mockFetch.mockReturnValue(Promise.resolve({
        ok: true,
        status: 204,
        statusText: 'No Content',
        json: () => Promise.resolve(undefined),
      } as Response))

      const payload = {
        subjectType: 'task',
        subjectId: 'task-1',
        kind: 'debrief',
        data: {
          outcome: 'completed',
          note: 'Finished early',
          contextId: 'ctx-1',
          priority: 'high',
          energy: 'medium',
        },
      }

      await observationService.record(payload)

      const url = mockFetch.mock.calls[0]![0] as string
      const options = mockFetch.mock.calls[0]![1] as RequestInit

      expect(url).toContain('/api/v1/observations')
      expect(options.method).toBe('POST')

      const body = JSON.parse(options.body as string)
      expect(body).toEqual(payload)
    })

    it('records skip observation with task metadata', async () => {
      mockFetch.mockReturnValue(Promise.resolve({
        ok: true,
        status: 204,
        statusText: 'No Content',
        json: () => Promise.resolve(undefined),
      } as Response))

      const payload = {
        subjectType: 'task',
        subjectId: 'task-1',
        kind: 'debrief',
        data: {
          outcome: 'skipped',
          contextId: 'ctx-1',
          priority: 'high',
          energy: 'high',
        },
      }

      await observationService.record(payload)

      const body = JSON.parse(
        (mockFetch.mock.calls[0]![1] as RequestInit).body as string,
      )
      expect(body.data.outcome).toBe('skipped')
      expect(body.data.contextId).toBe('ctx-1')
      expect(body.data.priority).toBe('high')
      expect(body.data.energy).toBe('high')
    })
  })
})
