import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useSearch } from '@/composables/useSearch'
import { makeTask, makeContext, makeTag, makeQueryResult } from '../helpers/testFactories'
import { taskService } from '@/services/taskService'
import { contextService } from '@/services/contextService'
import { tagService } from '@/services/tagService'

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

vi.mock('@/services/tagService', () => ({
  tagService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getByTask: vi.fn(),
    addToTask: vi.fn(),
    removeFromTask: vi.fn(),
    getByContext: vi.fn(),
    addToContext: vi.fn(),
    removeFromContext: vi.fn(),
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

describe('useSearch', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(tagService.list).mockResolvedValue(makeQueryResult([]))
  })

  it('activeTab defaults to all', () => {
    const { result, wrapper } = withSetup(() => useSearch())
    expect(result.activeTab.value).toBe('all')
    wrapper.unmount()
  })

  it('setTab changes activeTab', () => {
    const { result, wrapper } = withSetup(() => useSearch())
    result.setTab('tasks')
    expect(result.activeTab.value).toBe('tasks')
    result.setTab('contexts')
    expect(result.activeTab.value).toBe('contexts')
    wrapper.unmount()
  })

  it('filteredTasks filters by title match case-insensitive', async () => {
    const matchingTask = makeTask({ title: 'Fix login bug' })
    const nonMatchingTask = makeTask({ title: 'Update readme' })
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([matchingTask, nonMatchingTask]))

    const { result, wrapper } = withSetup(() => useSearch())

    await result.search()
    await nextTick()

    result.query.value = 'login'
    await nextTick()

    expect(result.filteredTasks.value).toHaveLength(1)
    expect(result.filteredTasks.value[0]!.id).toBe(matchingTask.id)

    wrapper.unmount()
  })

  it('filteredContexts filters by title and description match', async () => {
    const matchCtx = makeContext({ title: 'Project Alpha', description: 'Important project' })
    const descCtx = makeContext({ title: 'Other', description: 'alpha related work' })
    const noMatchCtx = makeContext({ title: 'Unrelated', description: 'nothing here' })
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([matchCtx, descCtx, noMatchCtx]))

    const { result, wrapper } = withSetup(() => useSearch())

    await result.search()
    await nextTick()

    result.query.value = 'alpha'
    await nextTick()

    expect(result.filteredContexts.value).toHaveLength(2)

    wrapper.unmount()
  })

  it('filteredTags filters by name match', async () => {
    const matchTag = makeTag({ name: 'urgent' })
    const noMatchTag = makeTag({ name: 'optional' })
    vi.mocked(tagService.list).mockResolvedValue(makeQueryResult([matchTag, noMatchTag]))

    const { result, wrapper } = withSetup(() => useSearch())

    await result.search()
    await nextTick()

    result.query.value = 'urg'
    await nextTick()

    expect(result.filteredTags.value).toHaveLength(1)
    expect(result.filteredTags.value[0]!.name).toBe('urgent')

    wrapper.unmount()
  })

  it('search() calls fetchList on all stores', async () => {
    const { result, wrapper } = withSetup(() => useSearch())

    await result.search()
    await nextTick()

    expect(taskService.list).toHaveBeenCalledTimes(1)
    expect(contextService.list).toHaveBeenCalledTimes(1)
    expect(tagService.list).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('totalResults sums all result counts', async () => {
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([makeTask({ title: 'test item' })]))
    vi.mocked(contextService.list).mockResolvedValue(
      makeQueryResult([makeContext({ title: 'test context' }), makeContext({ title: 'test ctx 2' })]),
    )
    vi.mocked(tagService.list).mockResolvedValue(makeQueryResult([makeTag({ name: 'test-tag' })]))

    const { result, wrapper } = withSetup(() => useSearch())

    await result.search()
    await nextTick()

    result.query.value = 'test'
    await nextTick()

    expect(result.totalResults.value).toBe(4)

    wrapper.unmount()
  })
})
