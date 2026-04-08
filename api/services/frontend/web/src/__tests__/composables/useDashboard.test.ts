import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
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
  },
}))

vi.mock('@/services/activityLogService', () => ({
  activityLogService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getStreaks: vi.fn(),
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

  it('fetches tasks, contexts, and activity logs on mount', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')
    const { activityLogService } = await import('@/services/activityLogService')

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = withSetup(() => useDashboard())
    await flushPromises()

    expect(taskService.list).toHaveBeenCalled()
    expect(contextService.list).toHaveBeenCalled()
    expect(activityLogService.list).toHaveBeenCalled()

    wrapper.unmount()
  })

  it('completionTrend returns 4 weekly buckets', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')
    const { activityLogService } = await import('@/services/activityLogService')

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.completionTrend.value).toHaveLength(4)
    expect(result.completionTrend.value[0]).toHaveProperty('weekLabel')
    expect(result.completionTrend.value[0]).toHaveProperty('completed')

    wrapper.unmount()
  })

  it('growingBacklogs groups open and blocked tasks by context', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')
    const { activityLogService } = await import('@/services/activityLogService')

    const ctxId = 'ctx-1'
    const tasks = [
      makeTask({ status: TaskStatus.Open, contextId: ctxId }),
      makeTask({ status: TaskStatus.Blocked, contextId: ctxId }),
      makeTask({ status: TaskStatus.Done, contextId: ctxId }), // should not count
    ]
    const contexts = [makeContext({ id: ctxId, status: ContextStatus.Active })]

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    const backlogs = result.growingBacklogs.value
    expect(backlogs.length).toBeGreaterThan(0)
    const entry = backlogs.find((b) => b.contextId === ctxId)
    expect(entry).toBeDefined()
    expect(entry!.total).toBe(2)
    expect(entry!.openCount).toBe(1)
    expect(entry!.blockedCount).toBe(1)

    wrapper.unmount()
  })

  it('repeatedlyDismissed returns dismissed tasks sorted by updatedAt descending', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')
    const { activityLogService } = await import('@/services/activityLogService')

    const tasks = [
      makeTask({ status: TaskStatus.Dismissed, updatedAt: new Date(Date.now() - 1000).toISOString() }),
      makeTask({ status: TaskStatus.Dismissed, updatedAt: new Date(Date.now() - 2000).toISOString() }),
      makeTask({ status: TaskStatus.Open }),
    ]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.repeatedlyDismissed.value).toHaveLength(2)

    wrapper.unmount()
  })

  it('inactiveContexts returns active contexts with no recent task activity', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')
    const { activityLogService } = await import('@/services/activityLogService')

    const activeCtxId = 'ctx-active-quiet'
    const contexts = [
      makeContext({ id: activeCtxId, status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Closed }),
    ]
    // No tasks and no activity logs → activeCtxId should appear as inactive
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    const inactive = result.inactiveContexts.value
    expect(inactive.some((c) => c.id === activeCtxId)).toBe(true)
    // Closed context should not appear
    expect(inactive.every((c) => c.status === ContextStatus.Active)).toBe(true)

    wrapper.unmount()
  })
})
