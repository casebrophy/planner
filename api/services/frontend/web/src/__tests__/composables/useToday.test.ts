import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useToday } from '@/composables/useToday'
import { makeTask } from '../helpers/testFactories'
import { TaskStatus } from '@/types'
import { fetchAllPages } from '@/services/fetchAllPages'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/fetchAllPages', () => ({
  fetchAllPages: vi.fn(),
}))

let mockContextItems: any[] = []

vi.mock('@/stores/contextStore', () => {
  return {
    useContextStore: () => ({
      get items() {
        return mockContextItems
      },
      set items(val) {
        mockContextItems = val
      },
      fetchAll: vi.fn().mockImplementation(() => {
        mockContextItems = []
        return Promise.resolve()
      }),
    }),
  }
})

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

function yesterday(): string {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toISOString()
}

function todayDate(): string {
  const d = new Date()
  d.setHours(12, 0, 0, 0)
  return d.toISOString()
}

function tomorrow(): string {
  const d = new Date()
  d.setDate(d.getDate() + 1)
  return d.toISOString()
}

describe('useToday', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('overdueTasks filters correctly', async () => {
    const overdueTask = makeTask({ status: TaskStatus.Open, dueDate: yesterday() })
    const doneTask = makeTask({ status: TaskStatus.Done, dueDate: yesterday() })
    const futureTask = makeTask({ status: TaskStatus.Open, dueDate: tomorrow() })
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [overdueTask, doneTask, futureTask], total: 3 })

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.overdueTasks.value).toHaveLength(1)
    expect(result.overdueTasks.value[0]!.id).toBe(overdueTask.id)

    wrapper.unmount()
  })

  it('dueTodayTasks filters correctly', async () => {
    const todayTask = makeTask({ status: TaskStatus.Open, dueDate: todayDate() })
    const pastTask = makeTask({ status: TaskStatus.Open, dueDate: yesterday() })
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [todayTask, pastTask], total: 2 })

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.dueTodayTasks.value).toHaveLength(1)
    expect(result.dueTodayTasks.value[0]!.id).toBe(todayTask.id)

    wrapper.unmount()
  })

  it('inProgressTasks filters by status', async () => {
    const ipTask = makeTask({ status: TaskStatus.Blocked })
    const todoTask = makeTask({ status: TaskStatus.Open })
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [ipTask, todoTask], total: 2 })

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.blockedTasks.value).toHaveLength(1)
    expect(result.blockedTasks.value[0]!.id).toBe(ipTask.id)

    wrapper.unmount()
  })

  it('contextMap returns an object', async () => {
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.contextMap.value).toBeDefined()
    expect(typeof result.contextMap.value).toBe('object')

    wrapper.unmount()
  })

  it('load fetches tasks and contexts', async () => {
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })

    const { wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(fetchAllPages).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('cancelled tasks are excluded from overdue', async () => {
    const cancelledTask = makeTask({ status: TaskStatus.Dismissed, dueDate: yesterday() })
    vi.mocked(fetchAllPages).mockResolvedValue({ items: [cancelledTask], total: 1 })

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.overdueTasks.value).toHaveLength(0)

    wrapper.unmount()
  })
})
