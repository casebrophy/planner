import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useContextBoard } from '@/composables/useContextBoard'
import { makeContext, makeQueryResult } from '../helpers/testFactories'
import { contextService } from '@/services/contextService'
import { ContextStatus } from '@/types'

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

describe('useContextBoard', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches contexts on mount', async () => {
    const contexts = [makeContext(), makeContext()]
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()
    await nextTick()

    expect(contextService.list).toHaveBeenCalledTimes(1)
    expect(result.contexts.value).toHaveLength(2)

    wrapper.unmount()
  })

  it('setFilter calls store.setFilter and re-fetches', async () => {
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([makeContext()]))

    result.setFilter({ status: ContextStatus.Active })
    await nextTick()

    expect(contextService.list).toHaveBeenCalledTimes(1)

    wrapper.unmount()
  })

  it('isEmpty is true when not loading and no items', async () => {
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()
    await nextTick()

    expect(result.isEmpty.value).toBe(true)

    wrapper.unmount()
  })

  it('contextsByStatus groups correctly', async () => {
    const contexts = [
      makeContext({ status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Active }),
      makeContext({ status: ContextStatus.Paused }),
    ]
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()
    await nextTick()

    expect(result.contextsByStatus.value[ContextStatus.Active]).toHaveLength(2)
    expect(result.contextsByStatus.value[ContextStatus.Paused]).toHaveLength(1)

    wrapper.unmount()
  })

  it('contextsByKind groups correctly', async () => {
    const { ContextKind } = await import('@/types')
    const contexts = [
      makeContext({ kind: ContextKind.Project }),
      makeContext({ kind: ContextKind.Project }),
      makeContext({ kind: ContextKind.Area }),
    ]
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()
    await nextTick()

    expect(result.contextsByKind.value[ContextKind.Project]).toHaveLength(2)
    expect(result.contextsByKind.value[ContextKind.Area]).toHaveLength(1)

    wrapper.unmount()
  })

  it('contextsByKind returns empty buckets for empty list', async () => {
    const { ContextKind } = await import('@/types')
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { result, wrapper } = withSetup(() => useContextBoard())
    await nextTick()
    await nextTick()

    expect(result.contextsByKind.value[ContextKind.Project]).toHaveLength(0)
    expect(result.contextsByKind.value[ContextKind.Area]).toHaveLength(0)

    wrapper.unmount()
  })
})
