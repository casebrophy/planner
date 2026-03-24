import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useClarification } from '@/composables/useClarification'
import { makeClarificationItem } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/clarificationService', () => ({
  clarificationService: {
    queryQueue: vi.fn(),
    countPending: vi.fn(),
    resolve: vi.fn(),
    snooze: vi.fn(),
    dismiss: vi.fn(),
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

describe('useClarification', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('fetches queue on mount', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    const items = [makeClarificationItem()]
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({ items, total: 1, page: 1, rowsPerPage: 50 })

    const { result, wrapper } = withSetup(() => useClarification())
    await nextTick()
    await nextTick()

    expect(clarificationService.queryQueue).toHaveBeenCalled()
    expect(result.items.value).toHaveLength(1)

    wrapper.unmount()
  })

  it('isEmpty is true when no pending items', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 })

    const { result, wrapper } = withSetup(() => useClarification())
    await nextTick()
    await nextTick()

    expect(result.isEmpty.value).toBe(true)

    wrapper.unmount()
  })

  it('refresh forces re-fetch', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 })

    const { result, wrapper } = withSetup(() => useClarification())
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({ items: [makeClarificationItem()], total: 1, page: 1, rowsPerPage: 50 })

    result.refresh()
    await nextTick()
    await nextTick()

    expect(clarificationService.queryQueue).toHaveBeenCalled()

    wrapper.unmount()
  })
})
