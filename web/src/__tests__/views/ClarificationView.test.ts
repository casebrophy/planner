import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import ClarificationView from '@/views/ClarificationView.vue'
import { makeClarificationItem } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/clarificationService', () => ({
  clarificationService: {
    queryQueue: vi.fn(),
    countPending: vi.fn().mockResolvedValue(0),
    resolve: vi.fn(),
    snooze: vi.fn(),
    dismiss: vi.fn(),
  },
}))

describe('ClarificationView', () => {
  it('renders page header', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({
      items: [makeClarificationItem(), makeClarificationItem()],
      total: 2,
      page: 1,
      rowsPerPage: 50,
    })

    const wrapper = mount(ClarificationView, {
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Clarifications')
    wrapper.unmount()
  })

  it('shows empty state when no items', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      rowsPerPage: 50,
    })

    const wrapper = mount(ClarificationView, {
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('All caught up')
    wrapper.unmount()
  })

  it('renders clarification card when items exist', async () => {
    const { clarificationService } = await import('@/services/clarificationService')
    const item = makeClarificationItem({ question: 'Is this task still relevant?' })
    vi.mocked(clarificationService.queryQueue).mockResolvedValue({
      items: [item],
      total: 1,
      page: 1,
      rowsPerPage: 50,
    })

    const wrapper = mount(ClarificationView, {
      global: { plugins: [createPinia()] },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Is this task still relevant?')
    wrapper.unmount()
  })
})
