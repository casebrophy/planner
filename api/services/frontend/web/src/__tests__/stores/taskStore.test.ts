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
