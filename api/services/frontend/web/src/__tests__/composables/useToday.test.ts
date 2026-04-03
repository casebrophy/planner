import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useToday } from '@/composables/useToday'
import { makeTask, makeContext, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus } from '@/types'
import { taskService } from '@/services/taskService'
import { contextService } from '@/services/contextService'

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
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([overdueTask, doneTask, futureTask]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

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
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([todayTask, pastTask]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

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
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([ipTask, todoTask]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.blockedTasks.value).toHaveLength(1)
    expect(result.blockedTasks.value[0]!.id).toBe(ipTask.id)

    wrapper.unmount()
  })

  it('contextMap maps context IDs to titles', async () => {
    const ctx = makeContext({ id: 'ctx-1', title: 'My Context' })
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([ctx]))

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.contextMap.value['ctx-1']).toBe('My Context')

    wrapper.unmount()
  })

  it('load fetches from both stores', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    expect(contextService.list).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('cancelled tasks are excluded from overdue', async () => {
    const cancelledTask = makeTask({ status: TaskStatus.Dismissed, dueDate: yesterday() })
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([cancelledTask]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useToday())
    await nextTick()
    await nextTick()

    expect(result.overdueTasks.value).toHaveLength(0)

    wrapper.unmount()
  })
})
