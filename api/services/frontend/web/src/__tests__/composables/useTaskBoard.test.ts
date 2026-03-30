import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useTaskBoard } from '@/composables/useTaskBoard'
import { makeTask, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus } from '@/types'
import { taskService } from '@/services/taskService'

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

// service.list is called with { page, rows, orderBy, filter }
interface ListCallArg {
  page: number
  rows: number
  orderBy: string
  filter: Record<string, unknown>
}

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

describe('useTaskBoard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches tasks on mount', async () => {
    const tasks = [makeTask(), makeTask()]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    expect(result.tasks.value).toHaveLength(2)
    expect(result.tasks.value[0]!.id).toBe(tasks[0]!.id)

    wrapper.unmount()
  })

  it('setFilter calls store.setFilter and re-fetches with the new filter', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([makeTask()]))

    result.setFilter({ status: TaskStatus.Todo })
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    const callArg = vi.mocked(taskService.list).mock.calls[0]![0] as ListCallArg
    expect(callArg.filter).toMatchObject({ status: TaskStatus.Todo })

    wrapper.unmount()
  })

  it('setOrder calls store.setOrder and re-fetches with the new orderBy', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    result.setOrder('title')
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    const callArg = vi.mocked(taskService.list).mock.calls[0]![0] as ListCallArg
    expect(callArg.orderBy).toBe('title')

    wrapper.unmount()
  })

  it('setPage calls store.setPage and re-fetches with the new page number', async () => {
    const tasks = Array.from({ length: 25 }, () => makeTask())
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks, 25))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks.slice(20), 25))

    result.setPage(2)
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    const callArg = vi.mocked(taskService.list).mock.calls[0]![0] as ListCallArg
    expect(callArg.page).toBe(2)

    wrapper.unmount()
  })

  it('refresh forces a fetch and updates tasks', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()
    await nextTick()

    vi.clearAllMocks()
    const refreshedTask = makeTask()
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([refreshedTask]))

    result.refresh()
    await nextTick()
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    expect(result.tasks.value).toHaveLength(1)
    expect(result.tasks.value[0]!.id).toBe(refreshedTask.id)

    wrapper.unmount()
  })

  it('isEmpty is true when not loading and no items', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()
    await nextTick()

    expect(result.loading.value).toBe(false)
    expect(result.tasks.value).toHaveLength(0)
    expect(result.isEmpty.value).toBe(true)

    wrapper.unmount()
  })

  it('isEmpty is false when items exist', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([makeTask(), makeTask()]))

    const { result, wrapper } = withSetup(() => useTaskBoard())
    await nextTick()
    await nextTick()

    expect(result.tasks.value.length).toBeGreaterThan(0)
    expect(result.isEmpty.value).toBe(false)

    wrapper.unmount()
  })
})
