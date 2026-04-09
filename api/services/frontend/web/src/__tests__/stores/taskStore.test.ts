import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTaskStore } from '@/stores/taskStore'
import { makeTask } from '../helpers/testFactories'
import { TaskStatus } from '@/types'
import type { Task, QueryResult } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

describe('taskStore computed extensions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  describe('tasksByStatus', () => {
    it('groups tasks by their status field', () => {
      const store = useTaskStore()
      store.items = [
        makeTask({ status: TaskStatus.Open }),
        makeTask({ status: TaskStatus.Open }),
        makeTask({ status: TaskStatus.Blocked }),
        makeTask({ status: TaskStatus.Done }),
      ]

      const groups = store.tasksByStatus
      expect(groups[TaskStatus.Open]).toHaveLength(2)
      expect(groups[TaskStatus.Blocked]).toHaveLength(1)
      expect(groups[TaskStatus.Done]).toHaveLength(1)
      expect(groups[TaskStatus.Dismissed]).toBeUndefined()
    })

    it('returns an empty object when items is empty', () => {
      const store = useTaskStore()
      store.items = []
      expect(store.tasksByStatus).toEqual({})
    })
  })

  describe('hasActiveFilter', () => {
    it('returns false when filter has no status, priority, or contextId', () => {
      const store = useTaskStore()
      store.filter = {}
      expect(store.hasActiveFilter).toBe(false)
    })

    it('returns true when filter has status set', () => {
      const store = useTaskStore()
      store.filter = { status: TaskStatus.Blocked }
      expect(store.hasActiveFilter).toBe(true)
    })

    it('returns true when filter has contextId set', () => {
      const store = useTaskStore()
      store.filter = { contextId: 'ctx-abc' }
      expect(store.hasActiveFilter).toBe(true)
    })
  })

  describe('habit/recurrence task properties', () => {
    it('tasks with recurrenceRule are identifiable as habits', () => {
      const store = useTaskStore()
      store.items = [
        makeTask({ recurrenceRule: 'daily' }),
        makeTask({ recurrenceRule: 'weekly' }),
        makeTask({}), // no recurrence
      ]

      const habits = store.items.filter((t) => t.recurrenceRule)
      expect(habits).toHaveLength(2)
    })

    it('tasks with trackOutcome flag are identifiable', () => {
      const store = useTaskStore()
      store.items = [
        makeTask({ trackOutcome: true, recurrenceRule: 'daily' }),
        makeTask({ trackOutcome: false }),
        makeTask({}), // trackOutcome undefined
      ]

      const tracked = store.items.filter((t) => t.trackOutcome)
      expect(tracked).toHaveLength(1)
    })

    it('recurrence parent relationship is accessible', () => {
      const store = useTaskStore()
      const parentId = 'parent-task-1'
      store.items = [
        makeTask({ id: parentId, recurrenceRule: 'daily' }),
        makeTask({ recurrenceParentId: parentId }),
        makeTask({ recurrenceParentId: parentId }),
      ]

      const children = store.items.filter((t) => t.recurrenceParentId === parentId)
      expect(children).toHaveLength(2)
    })
  })

  describe('fetchHabits', () => {
    it('fetches tasks with hasRecurrence filter and populates habits', async () => {
      const { taskService } = await import('@/services/taskService')
      const habitTasks = [
        makeTask({ recurrenceRule: 'daily' }),
        makeTask({ recurrenceRule: 'weekly' }),
      ]
      vi.mocked(taskService.list).mockResolvedValue({
        items: habitTasks,
        total: 2,
        page: 1,
        rowsPerPage: 100,
      })

      const store = useTaskStore()
      await store.fetchHabits()

      expect(taskService.list).toHaveBeenCalledWith(
        expect.objectContaining({
          filter: expect.objectContaining({ hasRecurrence: true }),
        }),
      )
      expect(store.habits).toHaveLength(2)
      expect(store.habits[0]!.recurrenceRule).toBe('daily')
    })

    it('sets habitsLoading during fetch', async () => {
      const { taskService } = await import('@/services/taskService')
      let resolveList!: (value: QueryResult<Task>) => void
      vi.mocked(taskService.list).mockReturnValue(
        new Promise((resolve) => {
          resolveList = resolve
        }),
      )

      const store = useTaskStore()
      const fetchPromise = store.fetchHabits()

      expect(store.habitsLoading).toBe(true)

      resolveList({ items: [], total: 0, page: 1, rowsPerPage: 100 })
      await fetchPromise

      expect(store.habitsLoading).toBe(false)
    })
  })

  describe('overdueCount', () => {
    it('counts tasks with a past dueDate that are not Done or Cancelled', () => {
      const store = useTaskStore()
      const pastDate = new Date(Date.now() - 86400000).toISOString() // yesterday
      store.items = [
        makeTask({ dueDate: pastDate, status: TaskStatus.Open }),
        makeTask({ dueDate: pastDate, status: TaskStatus.Blocked }),
        makeTask({ dueDate: pastDate, status: TaskStatus.Done }),
        makeTask({ dueDate: pastDate, status: TaskStatus.Dismissed }),
        makeTask({ status: TaskStatus.Open }), // no dueDate
      ]
      expect(store.overdueCount).toBe(2)
    })

    it('returns 0 when no tasks are overdue', () => {
      const store = useTaskStore()
      const futureDate = new Date(Date.now() + 86400000).toISOString() // tomorrow
      store.items = [
        makeTask({ dueDate: futureDate, status: TaskStatus.Open }),
        makeTask({ status: TaskStatus.Open }),
      ]
      expect(store.overdueCount).toBe(0)
    })
  })
})
