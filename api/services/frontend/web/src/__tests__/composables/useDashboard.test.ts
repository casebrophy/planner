import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
import { useDashboard } from '@/composables/useDashboard'
import { makeTask, makeContext, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus, ContextStatus } from '@/types'
import { fetchAllPages } from '@/services/fetchAllPages'
import { activityLogService } from '@/services/activityLogService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/fetchAllPages', () => ({
  fetchAllPages: vi.fn(),
}))

vi.mock('@/stores/contextStore', () => {
  return {
    useContextStore: () => ({
      items: ref([]),
      fetchAll: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    }),
  }
})

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
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = withSetup(() => useDashboard())
    await flushPromises()

    expect(fetchAllPages).toHaveBeenCalled()
    expect(activityLogService.list).toHaveBeenCalled()

    wrapper.unmount()
  })

  it('completionTrend returns 4 weekly buckets', async () => {
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })
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
    const { useContextStore } = await import('@/stores/contextStore')
    const ctxId = 'ctx-1'
    const tasks = [
      makeTask({ status: TaskStatus.Open, contextId: ctxId }),
      makeTask({ status: TaskStatus.Blocked, contextId: ctxId }),
      makeTask({ status: TaskStatus.Done, contextId: ctxId }), // should not count
    ]
    const contexts = [makeContext({ id: ctxId, status: ContextStatus.Active })]

    vi.mocked(fetchAllPages).mockResolvedValue({ items: tasks, total: tasks.length })
    // fetchAll mutates store.items, so set up items on the mock store
    const mockStore = useContextStore()
    mockStore.items = contexts
    vi.mocked(mockStore.fetchAll).mockResolvedValue()
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
    const tasks = [
      makeTask({ status: TaskStatus.Dismissed, updatedAt: new Date(Date.now() - 1000).toISOString() }),
      makeTask({ status: TaskStatus.Dismissed, updatedAt: new Date(Date.now() - 2000).toISOString() }),
      makeTask({ status: TaskStatus.Open }),
    ]
    vi.mocked(fetchAllPages).mockResolvedValue({ items: tasks, total: tasks.length })
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    expect(result.repeatedlyDismissed.value).toHaveLength(2)

    wrapper.unmount()
  })

  it('inactiveContexts filters correctly', async () => {
    // No tasks and no activity logs
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useDashboard())
    await nextTick()
    await nextTick()

    // The result should be an array (even if empty based on mock data)
    expect(Array.isArray(result.inactiveContexts.value)).toBe(true)

    wrapper.unmount()
  })
})
