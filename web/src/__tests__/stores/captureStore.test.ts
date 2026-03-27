import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useCaptureStore } from '@/stores/captureStore'
import { makeTask } from '../helpers/testFactories'
import { TaskPriority, TaskEnergy } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 20 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listEvents: vi.fn(),
    addEvent: vi.fn(),
  },
}))

describe('captureStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('defaults to task mode', () => {
    const store = useCaptureStore()
    expect(store.mode).toBe('task')
  })

  it('defaultTask returns a valid NewTask shape', () => {
    const store = useCaptureStore()
    const task = store.defaultTask()
    expect(task.title).toBe('')
    expect(task.priority).toBe(TaskPriority.Medium)
    expect(task.energy).toBe(TaskEnergy.Medium)
  })

  it('defaultContext returns a valid NewContext shape', () => {
    const store = useCaptureStore()
    const ctx = store.defaultContext()
    expect(ctx.title).toBe('')
    expect(ctx.description).toBe('')
  })

  it('submitTask sets submitting flag', async () => {
    const { taskService } = await import('@/services/taskService')
    const task = makeTask()
    vi.mocked(taskService.create).mockResolvedValue(task)

    const store = useCaptureStore()
    const promise = store.submitTask({ title: 'Test', description: '', priority: TaskPriority.Medium, energy: TaskEnergy.Medium })

    expect(store.submitting).toBe(true)
    await promise
    expect(store.submitting).toBe(false)
  })
})
