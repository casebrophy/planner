import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useContextDetail } from '@/composables/useContextDetail'
import { makeContext, makeContextEvent, makeTag, makeTask, makeQueryResult } from '../helpers/testFactories'
import { contextService } from '@/services/contextService'
import { tagService } from '@/services/tagService'
import { taskService } from '@/services/taskService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
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

vi.mock('@/services/taskService', () => ({
  taskService: {
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

describe('useContextDetail', () => {
  let wrapper: ReturnType<typeof mount>

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('loads context, events, tags, and tasks on mount', async () => {
    const ctx = makeContext()
    const event = makeContextEvent({ contextId: ctx.id })
    const tag = makeTag()

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(contextService.listEvents).mockResolvedValue({ items: [event], total: 1, page: 1, rowsPerPage: 50 })
    vi.mocked(tagService.getByContext).mockResolvedValue([tag])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(contextService.getById).toHaveBeenCalledWith(ctx.id)
    expect(contextService.listEvents).toHaveBeenCalledWith(ctx.id, expect.any(Object))
    expect(tagService.getByContext).toHaveBeenCalledWith(ctx.id)
    expect(setup.result.context.value).toEqual(ctx)
    expect(setup.result.tags.value).toEqual([tag])
  })

  it('linkedTasks filters tasks by contextId', async () => {
    const ctx = makeContext()
    const linkedTask = makeTask({ contextId: ctx.id })
    const otherTask = makeTask({ contextId: 'other-ctx' })

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(contextService.listEvents).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 })
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([linkedTask, otherTask]))

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(setup.result.linkedTasks.value).toHaveLength(1)
    expect(setup.result.linkedTasks.value[0]!.id).toBe(linkedTask.id)
  })

  it('update delegates to contextStore.update with contextId', async () => {
    const ctx = makeContext()
    const updatedCtx = { ...ctx, title: 'Updated' }

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(contextService.listEvents).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 })
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.update).mockResolvedValue(updatedCtx)

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.update({ title: 'Updated' })

    expect(contextService.update).toHaveBeenCalledWith(ctx.id, { title: 'Updated' })
  })

  it('addEvent delegates to contextStore.addEvent', async () => {
    const ctx = makeContext()
    const newEvent = makeContextEvent({ contextId: ctx.id })

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(contextService.listEvents).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 })
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.addEvent).mockResolvedValue(newEvent)

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.addEvent({ kind: 'note', content: 'test' })

    expect(contextService.addEvent).toHaveBeenCalledWith(ctx.id, { kind: 'note', content: 'test' })
  })
})
