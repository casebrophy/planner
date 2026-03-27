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
    listEvents: vi.fn(),
    addEvent: vi.fn(),
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
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    vi.mocked(taskService.list).mockReturnValue(new Promise(() => {}))
    vi.mocked(contextService.list).mockReturnValue(new Promise(() => {}))

    const wrapper = await mountView()
    expect(wrapper.findComponent({ name: 'LoadingSpinner' }).exists()).toBe(true)
    wrapper.unmount()
  })

  it('shows welcome empty state when no data', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult([]))

    const wrapper = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Welcome to Planner')
    wrapper.unmount()
  })

  it('renders summary cards with correct counts', async () => {
    const { taskService } = await import('@/services/taskService')
    const { contextService } = await import('@/services/contextService')

    const tasks = [
      makeTask({ status: TaskStatus.Todo }),
      makeTask({ status: TaskStatus.InProgress }),
    ]
    const contexts = [makeContext({ status: ContextStatus.Active })]

    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))
    vi.mocked(contextService.list).mockResolvedValue(makeQueryResult(contexts))

    const wrapper = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Total Tasks')
    expect(wrapper.text()).toContain('2')
    expect(wrapper.text()).toContain('Active Contexts')
    expect(wrapper.text()).toContain('1')
    wrapper.unmount()
  })
})
