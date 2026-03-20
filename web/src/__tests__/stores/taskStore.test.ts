import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTaskStore } from '@/stores/taskStore'
import { makeTask } from '../helpers/testFactories'
import { TaskStatus } from '@/types'

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
        makeTask({ status: TaskStatus.Todo }),
        makeTask({ status: TaskStatus.Todo }),
        makeTask({ status: TaskStatus.InProgress }),
        makeTask({ status: TaskStatus.Done }),
      ]

      const groups = store.tasksByStatus
      expect(groups[TaskStatus.Todo]).toHaveLength(2)
      expect(groups[TaskStatus.InProgress]).toHaveLength(1)
      expect(groups[TaskStatus.Done]).toHaveLength(1)
      expect(groups[TaskStatus.Cancelled]).toBeUndefined()
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
      store.filter = { status: TaskStatus.InProgress }
      expect(store.hasActiveFilter).toBe(true)
    })

    it('returns true when filter has contextId set', () => {
      const store = useTaskStore()
      store.filter = { contextId: 'ctx-abc' }
      expect(store.hasActiveFilter).toBe(true)
    })
  })

  describe('overdueCount', () => {
    it('counts tasks with a past dueDate that are not Done or Cancelled', () => {
      const store = useTaskStore()
      const pastDate = new Date(Date.now() - 86400000).toISOString() // yesterday
      store.items = [
        makeTask({ dueDate: pastDate, status: TaskStatus.Todo }),
        makeTask({ dueDate: pastDate, status: TaskStatus.InProgress }),
        makeTask({ dueDate: pastDate, status: TaskStatus.Done }),
        makeTask({ dueDate: pastDate, status: TaskStatus.Cancelled }),
        makeTask({ status: TaskStatus.Todo }), // no dueDate
      ]
      expect(store.overdueCount).toBe(2)
    })

    it('returns 0 when no tasks are overdue', () => {
      const store = useTaskStore()
      const futureDate = new Date(Date.now() + 86400000).toISOString() // tomorrow
      store.items = [
        makeTask({ dueDate: futureDate, status: TaskStatus.Todo }),
        makeTask({ status: TaskStatus.Todo }),
      ]
      expect(store.overdueCount).toBe(0)
    })
  })
})
