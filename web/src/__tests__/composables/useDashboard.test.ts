import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useDashboard } from '@/composables/useDashboard'
import { makeTask, makeContext, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus, ContextStatus } from '@/types'

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

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listEvents: vi.fn(),
    addEvent: vi.fn(),
  },
}))

function withSetup<T>(composable: () => T) {
  let result!: T
  const wrapper = mount(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
    { global: { plugins: [createPinia()] } },
  )
  return { result, wrapper }
}

describe('useDashboard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches tasks and contexts on mount', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(taskService.list).toHaveBeenCalled()
    expect(contextService.list).toHaveBeenCalled()

    wrapper.unmount()
  })

  it('computes task counts correctly', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const tasks = [
      makeTask({ status: TaskStatus.Todo }),
      makeTask({ status: TaskStatus.InProgress }),
      makeTask({ status: TaskStatus.Done }),
      makeTask({ status: TaskStatus.Todo, dueDate: pastDate }),
    ]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.taskCounts.value.total).toBe(4)
    expect(result.taskCounts.value.todo).toBe(2)
    expect(result.taskCounts.value.inProgress).toBe(1)
    expect(result.taskCounts.value.done).toBe(1)
    expect(result.taskCounts.value.overdue).toBe(1)

    wrapper.unmount()
  })

  it('computes context counts correctly', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const contexts = [
      makeContext({ status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Paused }),
    ]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.contextCounts.value.total).toBe(3)
    expect(result.contextCounts.value.active).toBe(2)
    expect(result.contextCounts.value.paused).toBe(1)

    wrapper.unmount()
  })

  it('recentTasks returns at most 5 most recently updated tasks', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const tasks = Array.from({ length: 8 }, (_, i) =>
      makeTask({ updatedAt: new Date(Date.now() - i * 60000).toISOString() }),
    )
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.recentTasks.value).toHaveLength(5)

    wrapper.unmount()
  })

  it('overdueTasks filters correctly', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const tasks = [
      makeTask({ status: TaskStatus.Todo, dueDate: pastDate }),
      makeTask({ status: TaskStatus.Done, dueDate: pastDate }),
      makeTask({ status: TaskStatus.Todo }),
    ]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.overdueTasks.value).toHaveLength(1)

    wrapper.unmount()
  })

  it('activeContexts filters to active status only', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const contexts = [
      makeContext({ status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Closed }),
    ]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.activeContexts.value).toHaveLength(1)

    wrapper.unmount()
  })
})
