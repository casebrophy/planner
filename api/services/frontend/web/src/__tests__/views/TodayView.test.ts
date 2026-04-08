import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { nextTick, ref, computed } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TodayView from '@/views/TodayView.vue'
import { makeTask, makeQueryResult } from '../helpers/testFactories'
import { TaskStatus } from '@/types'
import type { Task, Context, QueryResult } from '@/types'

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
  },
}))

vi.mock('@/composables/useDailyPlan', () => ({
  useDailyPlan: () => ({
    plan: ref(null),
    loading: ref(false),
    generating: ref(false),
    groupedItems: computed(() => []),
    taskMap: computed(() => ({})),
    completedCount: computed(() => 0),
    totalCount: computed(() => 0),
    refresh: vi.fn(),
    regenerate: vi.fn(),
    completeItem: vi.fn(),
    dismissItem: vi.fn(),
    reorderItem: vi.fn(),
  }),
}))

import { taskService } from '@/services/taskService'
import { contextService } from '@/services/contextService'

function yesterday(): string {
  const d = new Date()
  d.setDate(d.getDate() - 1)
  return d.toISOString()
}

function todayDate(): string {
  const d = new Date()
  d.setHours(12, 0, 0, 0)
  return d.toISOString()
}

async function mountView(
  taskData: QueryResult<Task> = makeQueryResult<Task>([]),
  contextData: QueryResult<Context> = makeQueryResult<Context>([]),
) {
  vi.mocked(taskService.list).mockResolvedValue(taskData)
  vi.mocked(contextService.list).mockResolvedValue(contextData)

  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'today', component: TodayView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
    ],
  })
  await router.push('/')
  await router.isReady()

  const wrapper = mount(TodayView, {
    global: {
      plugins: [createPinia(), router],
    },
  })

  return { wrapper, router }
}

describe('TodayView', () => {
  it('shows loading spinner initially', async () => {
    vi.mocked(taskService.list).mockReturnValue(new Promise(() => {}))
    vi.mocked(contextService.list).mockReturnValue(new Promise(() => {}))

    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', name: 'today', component: TodayView },
        { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
      ],
    })
    await router.push('/')
    await router.isReady()

    const wrapper = mount(TodayView, {
      global: { plugins: [createPinia(), router] },
    })

    await nextTick()

    // LoadingSpinner renders an svg with animate-spin class
    expect(wrapper.find('.animate-spin').exists()).toBe(true)
    wrapper.unmount()
  })

  it('shows empty state when no tasks', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.findComponent({ name: 'EmptyState' }).exists()).toBe(true)
    expect(wrapper.text()).toContain('All clear!')
    wrapper.unmount()
  })

  it('renders overdue section when overdue tasks exist', async () => {
    const overdueTask = makeTask({ status: TaskStatus.Open, dueDate: yesterday(), title: 'Overdue Task' })
    const { wrapper } = await mountView(makeQueryResult([overdueTask]))
    await flushPromises()

    expect(wrapper.text()).toContain('Overdue')
    expect(wrapper.text()).toContain('Overdue Task')
    wrapper.unmount()
  })

  it('renders due today section', async () => {
    const todayTask = makeTask({ status: TaskStatus.Open, dueDate: todayDate(), title: 'Today Task' })
    const { wrapper } = await mountView(makeQueryResult([todayTask]))
    await flushPromises()

    expect(wrapper.text()).toContain('Due Today')
    expect(wrapper.text()).toContain('Today Task')
    wrapper.unmount()
  })

  it('renders in progress section', async () => {
    const ipTask = makeTask({ status: TaskStatus.Blocked, title: 'IP Task' })
    const { wrapper } = await mountView(makeQueryResult([ipTask]))
    await flushPromises()

    expect(wrapper.text()).toContain('Blocked')
    expect(wrapper.text()).toContain('IP Task')
    wrapper.unmount()
  })
})
