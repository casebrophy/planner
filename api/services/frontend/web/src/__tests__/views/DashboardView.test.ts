import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import DashboardView from '@/views/DashboardView.vue'
import { makeTask, makeContext, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus, ContextStatus } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/fetchAllPages', () => ({
  fetchAllPages: vi.fn(),
}))

vi.mock('@/stores/contextStore', () => {
  const { ref } = require('vue')
  return {
    useContextStore: () => ({
      items: ref([]),
      fetchAll: vi.fn().mockResolvedValue({ items: [], total: 0 }),
    }),
  }
})

vi.mock('@/services/activityLogService', () => ({
  activityLogService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getStreaks: vi.fn(),
  },
}))

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: DashboardView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
      { path: '/contexts/:id', name: 'context-detail', component: { template: '<div />' } },
    ],
  })
  router.push('/')
  await router.isReady()

  return mount(DashboardView, {
    global: { plugins: [createPinia(), router] },
  })
}

describe('DashboardView', () => {
  it('shows loading spinner initially', async () => {
    const { fetchAllPages } = await import('@/services/fetchAllPages')
    const { activityLogService } = await import('@/services/activityLogService')

    vi.mocked(fetchAllPages).mockReturnValue(new Promise(() => {}))
    vi.mocked(activityLogService.list).mockReturnValue(new Promise(() => {}))

    const wrapper = await mountView()
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
    wrapper.unmount()
  })

  it('shows weekly health sections when no data', async () => {
    const { fetchAllPages } = await import('@/services/fetchAllPages')
    const { activityLogService } = await import('@/services/activityLogService')

    vi.mocked(fetchAllPages).mockResolvedValue({ items: [], total: 0 })
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const wrapper = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Weekly Health')
    expect(wrapper.text()).toContain('No open or blocked tasks found.')
    wrapper.unmount()
  })

  it('renders growing backlogs section with open and blocked tasks', async () => {
    const { fetchAllPages } = await import('@/services/fetchAllPages')
    const { useContextStore } = await import('@/stores/contextStore')
    const { activityLogService } = await import('@/services/activityLogService')

    const ctx = makeContext({ status: ContextStatus.Active })
    const tasks = [
      makeTask({ status: TaskStatus.Open, contextId: ctx.id }),
      makeTask({ status: TaskStatus.Blocked, contextId: ctx.id }),
    ]

    vi.mocked(fetchAllPages).mockResolvedValue({ items: tasks, total: tasks.length })
    // fetchAll mutates store.items, so set up items on the mock store
    const mockStore = useContextStore()
    mockStore.items = [ctx]
    vi.mocked(mockStore.fetchAll).mockResolvedValue()
    vi.mocked(activityLogService.list).mockResolvedValue(makeQueryResult([]))

    const wrapper = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Growing Backlogs')
    expect(wrapper.text()).toContain('1 open')
    expect(wrapper.text()).toContain('1 blocked')
    wrapper.unmount()
  })
})
