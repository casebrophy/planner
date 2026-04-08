import { describe, it, expect, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import CaptureView from '@/views/CaptureView.vue'
import { makeTask } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 20 }),
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

async function mountView() {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'capture', component: CaptureView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
      { path: '/contexts/:id', name: 'context-detail', component: { template: '<div />' } },
    ],
  })
  router.push('/')
  await router.isReady()

  return mount(CaptureView, {
    global: { plugins: [createPinia(), router] },
  })
}

describe('CaptureView', () => {
  it('renders page header', async () => {
    const wrapper = await mountView()
    expect(wrapper.text()).toContain('Quick Capture')
    wrapper.unmount()
  })

  it('defaults to task mode', async () => {
    const wrapper = await mountView()
    const taskBtn = wrapper.findAll('button').find(b => b.text() === 'Task')!
    expect(taskBtn.classes()).toContain('bg-blue-600')
    wrapper.unmount()
  })

  it('switches to context mode', async () => {
    const wrapper = await mountView()
    const contextBtn = wrapper.findAll('button').find(b => b.text() === 'Context')!
    await contextBtn.trigger('click')
    expect(contextBtn.classes()).toContain('bg-blue-600')
    wrapper.unmount()
  })

  it('submit button is disabled when title is empty', async () => {
    const wrapper = await mountView()
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })

  it('submit button is enabled when title is filled', async () => {
    const wrapper = await mountView()
    await wrapper.find('input[type="text"]').setValue('My task')
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeUndefined()
    wrapper.unmount()
  })

  it('submits task and calls taskService.create', async () => {
    const { taskService } = await import('@/services/taskService')
    const task = makeTask()
    vi.mocked(taskService.create).mockResolvedValue(task)

    const wrapper = await mountView()
    await wrapper.find('input[type="text"]').setValue('New task')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(taskService.create).toHaveBeenCalled()
    wrapper.unmount()
  })
})
