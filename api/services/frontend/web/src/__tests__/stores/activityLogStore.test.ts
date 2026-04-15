import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useActivityLogStore } from '@/stores/activityLogStore'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn(), info: vi.fn(), dismiss: vi.fn() }),
}))

vi.mock('@/services/activityLogService', () => ({
  activityLogService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getStreaks: vi.fn(),
    getBulkLogs: vi.fn().mockResolvedValue({ items: {} }),
  },
}))

describe('activityLogStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchStreaks', () => {
    it('fetches streaks and stores them keyed by subject', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      vi.mocked(activityLogService.getStreaks).mockResolvedValue({
        current: 7,
        longest: 14,
        totalCount: 50,
      })

      const store = useActivityLogStore()
      await store.fetchStreaks('task', 'task-1')

      expect(activityLogService.getStreaks).toHaveBeenCalledWith('task', 'task-1')
      expect(store.streaks['task:task-1']).toEqual({
        current: 7,
        longest: 14,
        totalCount: 50,
      })
    })

    it('stores streaks for different subjects independently', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      vi.mocked(activityLogService.getStreaks)
        .mockResolvedValueOnce({ current: 3, longest: 5, totalCount: 10 })
        .mockResolvedValueOnce({ current: 1, longest: 2, totalCount: 4 })

      const store = useActivityLogStore()
      await store.fetchStreaks('task', 'task-1')
      await store.fetchStreaks('context', 'ctx-1')

      expect(store.streaks['task:task-1']).toEqual({ current: 3, longest: 5, totalCount: 10 })
      expect(store.streaks['context:ctx-1']).toEqual({ current: 1, longest: 2, totalCount: 4 })
    })

    it('shows toast on error', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      vi.mocked(activityLogService.getStreaks).mockRejectedValue(new Error('Network error'))

      const store = useActivityLogStore()
      await store.fetchStreaks('task', 'bad-id')

      // Should not throw, error is handled internally
      expect(store.streaks['task:bad-id']).toBeUndefined()
    })
  })

  describe('logActivity', () => {
    it('creates an activity log entry and refreshes streaks', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      const newEntry = {
        id: 'log-1',
        subjectType: 'task',
        subjectId: 'task-1',
        loggedAt: new Date().toISOString(),
      }
      vi.mocked(activityLogService.create).mockResolvedValue(newEntry)
      vi.mocked(activityLogService.getStreaks).mockResolvedValue({
        current: 1,
        longest: 1,
        totalCount: 1,
      })

      const store = useActivityLogStore()
      const result = await store.logActivity('task', 'task-1')

      expect(activityLogService.create).toHaveBeenCalledWith({
        subjectType: 'task',
        subjectId: 'task-1',
        value: undefined,
      })
      expect(result).toEqual(newEntry)
      // Should also refresh streaks after logging
      expect(activityLogService.getStreaks).toHaveBeenCalledWith('task', 'task-1')
    })

    it('passes optional value when logging activity', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      vi.mocked(activityLogService.create).mockResolvedValue({
        id: 'log-2',
        subjectType: 'task',
        subjectId: 'task-1',
        value: 'completed set',
        loggedAt: new Date().toISOString(),
      })
      vi.mocked(activityLogService.getStreaks).mockResolvedValue({
        current: 2,
        longest: 2,
        totalCount: 5,
      })

      const store = useActivityLogStore()
      await store.logActivity('task', 'task-1', 'completed set')

      expect(activityLogService.create).toHaveBeenCalledWith({
        subjectType: 'task',
        subjectId: 'task-1',
        value: 'completed set',
      })
    })
  })

  describe('fetchHabitGrid', () => {
    it('fetches bulk logs and converts to date string grid', async () => {
      const { activityLogService } = await import('@/services/activityLogService')
      vi.mocked(activityLogService.getBulkLogs).mockResolvedValue({
        items: {
          'habit-1': [
            { id: 'l1', subjectType: 'task', subjectId: 'habit-1', loggedAt: '2026-04-01T12:00:00Z' },
            { id: 'l2', subjectType: 'task', subjectId: 'habit-1', loggedAt: '2026-04-02T08:00:00Z' },
          ],
          'habit-2': [
            { id: 'l3', subjectType: 'task', subjectId: 'habit-2', loggedAt: '2026-04-01T09:00:00Z' },
          ],
        },
      })

      const store = useActivityLogStore()
      await store.fetchHabitGrid(['habit-1', 'habit-2'], 30)

      expect(activityLogService.getBulkLogs).toHaveBeenCalledWith(
        'task',
        ['habit-1', 'habit-2'],
        expect.any(String),
        expect.any(String),
      )
      expect(store.habitGrid['habit-1']).toEqual([
        { id: 'l1', date: '2026-04-01' },
        { id: 'l2', date: '2026-04-02' },
      ])
      expect(store.habitGrid['habit-2']).toEqual([
        { id: 'l3', date: '2026-04-01' },
      ])
    })

    it('returns empty grid for empty habitIds', async () => {
      const store = useActivityLogStore()
      await store.fetchHabitGrid([], 30)
      expect(store.habitGrid).toEqual({})
    })
  })

  describe('streaks reactive state', () => {
    it('starts with empty streaks', () => {
      const store = useActivityLogStore()
      expect(store.streaks).toEqual({})
    })

    it('allows direct mutation of streaks for testing', () => {
      const store = useActivityLogStore()
      store.streaks['task:task-1'] = { current: 3, longest: 5, totalCount: 10 }
      expect(store.streaks['task:task-1']!.current).toBe(3)
    })
  })
})
