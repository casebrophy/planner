import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createWebHistory } from 'vue-router'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { makeTask, makeContext } from '../../helpers/testFactories'
import { useContextStore } from '@/stores/contextStore'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 20 }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    listEvents: vi.fn(),
    addEvent: vi.fn(),
  },
}))

const router = createRouter({
  history: createWebHistory(),
  routes: [{ path: '/contexts/:id', name: 'context-detail', component: { template: '<div />' } }],
})

function mountCard(props: NonNullable<NonNullable<Parameters<typeof mount>[1]>['props']>) {
  return mount(TaskCard, {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    props: props as any,
    global: {
      plugins: [createPinia(), router],
    },
  })
}

describe('TaskCard', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('renders task title', () => {
    const task = makeTask({ title: 'Buy groceries' })
    const wrapper = mountCard({ task })
    expect(wrapper.text()).toContain('Buy groceries')
  })

  it('renders task description when present', () => {
    const task = makeTask({ description: 'Milk and bread' })
    const wrapper = mountCard({ task })
    expect(wrapper.text()).toContain('Milk and bread')
  })

  it('emits click with task id', async () => {
    const task = makeTask()
    const wrapper = mountCard({ task })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toEqual([[task.id]])
  })

  it('shows overdue styling for past due tasks', () => {
    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const task = makeTask({ dueDate: pastDate, status: 'open' })
    const wrapper = mountCard({ task })
    expect(wrapper.find('.text-red-400').exists()).toBe(true)
  })

  it('does not show overdue for completed tasks', () => {
    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const task = makeTask({ dueDate: pastDate, status: 'done' })
    const wrapper = mountCard({ task })
    expect(wrapper.find('.text-red-400').exists()).toBe(false)
  })

  it('shows context chip when task has contextId and context exists in store', () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const ctx = makeContext({ id: 'ctx-1', title: 'My Project' })
    const store = useContextStore()
    store.items.push(ctx)

    const task = makeTask({ contextId: 'ctx-1' })
    const wrapper = mount(TaskCard, {
      props: { task },
      global: { plugins: [pinia, router] },
    })

    expect(wrapper.text()).toContain('My Project')
  })

  it('does not show context chip when task has no contextId', () => {
    const task = makeTask({ contextId: undefined })
    const wrapper = mountCard({ task })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(0)
  })

  it('does not show context chip when contextId not found in store', () => {
    const task = makeTask({ contextId: 'nonexistent-id' })
    const wrapper = mountCard({ task })
    const buttons = wrapper.findAll('button')
    expect(buttons).toHaveLength(0)
  })

  it('chip click navigates to context and does not emit card click', async () => {
    const pinia = createPinia()
    setActivePinia(pinia)
    const ctx = makeContext({ id: 'ctx-nav', title: 'Nav Context' })
    const store = useContextStore()
    store.items.push(ctx)

    const pushSpy = vi.spyOn(router, 'push').mockResolvedValue(undefined as never)

    const task = makeTask({ contextId: 'ctx-nav' })
    const wrapper = mount(TaskCard, {
      props: { task },
      global: { plugins: [pinia, router] },
    })

    const chip = wrapper.find('button')
    await chip.trigger('click')

    expect(pushSpy).toHaveBeenCalledWith('/contexts/ctx-nav')
    // Card click should NOT have fired (click.stop)
    expect(wrapper.emitted('click')).toBeFalsy()

    pushSpy.mockRestore()
  })
})
