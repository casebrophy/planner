import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ContextBoardView from '@/views/ContextBoardView.vue'
import { makeContext, makeQueryResult } from '../helpers/testFactories'
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

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'contexts', component: ContextBoardView },
      { path: '/contexts/:id', name: 'context-detail', component: { template: '<div />' } },
    ],
  })
  router.push('/')
  await router.isReady()

  return {
    wrapper: mount(ContextBoardView, {
      global: {
        plugins: [createPinia(), router],
        stubs: { Teleport: true },
      },
    }),
    router,
  }
}

describe('ContextBoardView', () => {
  it('renders page header with context count', async () => {
    const { contextService } = await import('@/services/contextService')
    const contexts = [makeContext(), makeContext(), makeContext()]
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Contexts')
    expect(wrapper.text()).toContain('3 contexts')
    wrapper.unmount()
  })

  it('shows empty state when no contexts', async () => {
    const { contextService } = await import('@/services/contextService')
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('No contexts found')
    wrapper.unmount()
  })

  it('renders kanban columns', async () => {
    const { contextService } = await import('@/services/contextService')
    const contexts = [
      makeContext({ status: ContextStatus.Active, title: 'Active ctx' }),
      makeContext({ status: ContextStatus.Paused, title: 'Paused ctx' }),
    ]
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Active')
    expect(wrapper.text()).toContain('Paused')
    expect(wrapper.text()).toContain('Closed')
    wrapper.unmount()
  })

  it('has New Context button', async () => {
    const { contextService } = await import('@/services/contextService')
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = await mountView()
    await flushPromises()

    const newBtn = wrapper.findAll('button').find(b => b.text() === 'New Context')
    expect(newBtn).toBeTruthy()
    wrapper.unmount()
  })
})
