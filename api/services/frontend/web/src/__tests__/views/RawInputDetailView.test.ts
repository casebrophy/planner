import { describe, it, expect, beforeEach, vi } from 'vitest'
import { reactive } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import RawInputDetailView from '@/views/RawInputDetailView.vue'
import type { RawInput } from '@/types/rawinput'

const mockStore = reactive({
  selectedItem: null as RawInput | null,
  loading: false,
  fetchById: vi.fn().mockResolvedValue(undefined),
  reprocess: vi.fn().mockResolvedValue(undefined),
})

vi.mock('@/stores/rawinputStore', () => ({
  useRawInputStore: () => mockStore,
}))

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

function makeRawInput(overrides: Partial<RawInput> = {}): RawInput {
  return {
    id: 'ri-1',
    sourceType: 'voice',
    status: 'processed',
    rawContent: 'buy groceries and call dentist',
    retryCount: 0,
    maxRetries: 5,
    createdAt: '2026-04-07T10:00:00Z',
    ...overrides,
  }
}

async function mountView(id: string = 'ri-1') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/ingest-queue', name: 'ingest-queue', component: { template: '<div />' } },
      { path: '/ingest-queue/:id', name: 'rawinput-detail', component: RawInputDetailView },
    ],
  })
  router.push(`/ingest-queue/${id}`)
  await router.isReady()

  const wrapper = mount(RawInputDetailView, {
    global: {
      plugins: [createPinia(), router],
      stubs: {
        LoadingSpinner: true,
      },
    },
  })
  await flushPromises()
  return { wrapper, router }
}

describe('RawInputDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockStore.selectedItem = null
    mockStore.loading = false
  })

  it('calls fetchById on mount', async () => {
    const { wrapper } = await mountView('ri-1')
    expect(mockStore.fetchById).toHaveBeenCalledWith('ri-1')
    wrapper.unmount()
  })

  it('shows loading spinner when loading with no item', async () => {
    mockStore.loading = true
    const { wrapper } = await mountView()
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
    wrapper.unmount()
  })

  it('shows raw input header with source type and status', async () => {
    mockStore.selectedItem = makeRawInput()
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('voice')
    expect(wrapper.text()).toContain('processed')
    wrapper.unmount()
  })

  it('shows error block when error exists', async () => {
    mockStore.selectedItem = makeRawInput({ status: 'failed', error: 'extraction timeout' })
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Error')
    expect(wrapper.text()).toContain('extraction timeout')
    wrapper.unmount()
  })

  it('shows pipeline steps when result exists', async () => {
    mockStore.selectedItem = makeRawInput({
      result: {
        sanitize: { status: 'completed', detail: { pii_findings: 0 } },
        extraction: { status: 'completed', detail: { action_items: 2, events: 1, notes: 0 } },
        contextMatch: { status: 'completed', detail: { context_id: 'ctx-1' } },
        tasks: { status: 'completed', detail: { count: 2, ids: ['t1', 't2'] } },
        events: { status: 'completed', detail: { count: 1, ids: ['e1'] } },
        notes: { status: 'completed', detail: { count: 0, ids: [] } },
      },
    })
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Pipeline Steps')
    expect(wrapper.text()).toContain('Sanitize')
    expect(wrapper.text()).toContain('AI Extraction')
    expect(wrapper.text()).toContain('Context Match')
    expect(wrapper.text()).toContain('Task Creation')
    expect(wrapper.text()).toContain('Event Creation')
    expect(wrapper.text()).toContain('Note Creation')
    wrapper.unmount()
  })

  it('shows failed extraction step with error detail', async () => {
    mockStore.selectedItem = makeRawInput({
      result: {
        sanitize: { status: 'completed', detail: { pii_findings: 0 } },
        extraction: { status: 'failed', detail: { error: 'claude cli timeout' } },
      },
    })
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('claude cli timeout')
    wrapper.unmount()
  })

  it('shows fallback message for items without result', async () => {
    mockStore.selectedItem = makeRawInput({ result: undefined })
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('No pipeline result recorded')
    wrapper.unmount()
  })

  it('toggles raw content on click', async () => {
    mockStore.selectedItem = makeRawInput()
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('buy groceries')
    const toggleBtn = wrapper.findAll('button').find(b => b.text().includes('Raw Content'))
    expect(toggleBtn).toBeDefined()
    await toggleBtn!.trigger('click')
    expect(wrapper.text()).toContain('buy groceries')
    wrapper.unmount()
  })

  it('shows reprocess button for failed items', async () => {
    mockStore.selectedItem = makeRawInput({ status: 'failed' })
    const { wrapper } = await mountView()
    await flushPromises()
    const reprocessBtn = wrapper.findAll('button').find(b => b.text().includes('Reprocess'))
    expect(reprocessBtn).toBeDefined()
    await reprocessBtn!.trigger('click')
    expect(mockStore.reprocess).toHaveBeenCalledWith('ri-1')
    wrapper.unmount()
  })

  it('shows not found when no item loaded', async () => {
    mockStore.selectedItem = null
    mockStore.loading = false
    const { wrapper } = await mountView()
    await flushPromises()
    expect(wrapper.text()).toContain('Raw input not found')
    wrapper.unmount()
  })
})
