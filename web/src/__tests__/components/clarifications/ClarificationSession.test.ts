import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { nextTick } from 'vue'
import ClarificationSession from '@/components/clarifications/ClarificationSession.vue'
import { useClarificationStore } from '@/stores/clarificationStore'
import { makeClarificationItem } from '../../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/clarificationService', () => ({
  clarificationService: {
    queryQueue: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
    countPending: vi.fn().mockResolvedValue(0),
    resolve: vi.fn().mockResolvedValue({}),
    snooze: vi.fn().mockResolvedValue({}),
    dismiss: vi.fn().mockResolvedValue({}),
  },
}))

describe('ClarificationSession', () => {
  function mountSession() {
    const pinia = createPinia()
    setActivePinia(pinia)
    return mount(ClarificationSession, {
      global: { plugins: [pinia] },
    })
  }

  it('shows empty state when no items', async () => {
    const wrapper = mountSession()
    const store = useClarificationStore()
    store.items = []
    store.loading = false
    await nextTick()
    expect(wrapper.text()).toContain('All caught up')
  })

  it('shows loading spinner when loading', async () => {
    const wrapper = mountSession()
    const store = useClarificationStore()
    store.loading = true
    store.items = []
    await nextTick()
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
  })

  it('renders current card when items exist', async () => {
    const wrapper = mountSession()
    const store = useClarificationStore()
    store.items = [makeClarificationItem({ question: 'Test question?' })]
    store.loading = false
    store.currentIndex = 0
    await nextTick()
    expect(wrapper.text()).toContain('Test question?')
  })

  it('shows progress indicator', async () => {
    const wrapper = mountSession()
    const store = useClarificationStore()
    store.items = [makeClarificationItem(), makeClarificationItem()]
    store.loading = false
    store.currentIndex = 0
    await nextTick()
    expect(wrapper.text()).toContain('1 of 2')
  })
})
