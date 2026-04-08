import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TaskBoardView from '@/views/TaskBoardView.vue'
import { makeTask, makeQueryResult } from '../helpers/testFactories'

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
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/services/tagService', () => ({
  tagService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getByTask: vi.fn().mockResolvedValue([]),
    addToTask: vi.fn(),
    removeFromTask: vi.fn(),
    getByContext: vi.fn(),
    addToContext: vi.fn(),
    removeFromContext: vi.fn(),
  },
}))

vi.mock('@/services/threadService', () => ({
  threadService: {
    queryBySubject: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }),
  },
}))

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'tasks', component: TaskBoardView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
    ],
  })
  router.push('/')
  await router.isReady()

  return {
    wrapper: mount(TaskBoardView, {
      global: {
        plugins: [createPinia(), router],
        stubs: { Teleport: true },
      },
    }),
    router,
  }
}

describe('TaskBoardView', () => {
  it('renders page header with task count', async () => {
    const { taskService } = await import('@/services/taskService')
    const tasks = [makeTask(), makeTask()]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Tasks')
    expect(wrapper.text()).toContain('2 tasks')
    wrapper.unmount()
  })

  it('shows empty state when no tasks', async () => {
    const { taskService } = await import('@/services/taskService')
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('No tasks found')
    wrapper.unmount()
  })

  it('renders task cards', async () => {
    const { taskService } = await import('@/services/taskService')
    const tasks = [makeTask({ title: 'Buy groceries' }), makeTask({ title: 'Fix bug' })]
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult(tasks))

    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Buy groceries')
    expect(wrapper.text()).toContain('Fix bug')
    wrapper.unmount()
  })

  it('has New Task button', async () => {
    const { taskService } = await import('@/services/taskService')
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = await mountView()
    await flushPromises()

    const newBtn = wrapper.findAll('button').find(b => b.text() === 'New Task')
    expect(newBtn).toBeTruthy()
    wrapper.unmount()
  })

  it('has Refresh button', async () => {
    const { taskService } = await import('@/services/taskService')
    vi.mocked(taskService.list).mockResolvedValue(makeQueryResult([]))

    const { wrapper } = await mountView()
    await flushPromises()

    const refreshBtn = wrapper.findAll('button').find(b => b.text() === 'Refresh')
    expect(refreshBtn).toBeTruthy()
    wrapper.unmount()
  })
})
