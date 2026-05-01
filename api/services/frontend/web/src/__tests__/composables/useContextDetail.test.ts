import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useContextDetail } from '@/composables/useContextDetail'
import { makeContext, makeTag, makeTask, makeQueryResult } from '../helpers/testFactories'
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

  it('loads context, tags, and tasks on mount', async () => {
    const ctx = makeContext()
    const tag = makeTag()

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(tagService.getByContext).mockResolvedValue([tag])
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useContextDetail(ctx.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()
    await nextTick()

    expect(contextService.getById).toHaveBeenCalledWith(ctx.id)
    expect(tagService.getByContext).toHaveBeenCalledWith(ctx.id)
    expect(taskService.list).toHaveBeenCalledWith(expect.objectContaining({ filter: { contextId: ctx.id } }))
    expect(setup.result.context.value).toEqual(ctx)
    expect(setup.result.tags.value).toEqual([tag])
  })

  it('linkedTasks filters tasks by contextId', async () => {
    const ctx = makeContext()
    const linkedTask = makeTask({ contextId: ctx.id })

    vi.mocked(contextService.getById).mockResolvedValue(ctx)
    vi.mocked(tagService.getByContext).mockResolvedValue([])
    vi.mocked(taskService.list).mockImplementation(async (opts) => {
      if (opts?.filter?.contextId === ctx.id) {
        return makeQueryResult([linkedTask])
      }
      return makeQueryResult([])
    })

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
})
