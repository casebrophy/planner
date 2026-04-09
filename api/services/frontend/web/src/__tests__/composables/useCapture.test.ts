/* eslint-disable vue/one-component-per-file */
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useCapture } from '@/composables/useCapture'
import { makeTask } from '../helpers/testFactories'
import { createRouter, createMemoryHistory } from 'vue-router'

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

function withSetup<T>(composable: () => T) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', component: defineComponent({ template: '<div />' }) },
      { path: '/tasks/:id', name: 'task-detail', component: defineComponent({ template: '<div />' }) },
      { path: '/contexts/:id', name: 'context-detail', component: defineComponent({ template: '<div />' }) },
    ],
  })

  let result!: T
  const wrapper = mount(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
    { global: { plugins: [createPinia(), router] } },
  )
  return { result, wrapper, router }
}

describe('useCapture', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('defaults to task mode', () => {
    const { result, wrapper } = withSetup(() => useCapture())
    expect(result.mode.value).toBe('task')
    wrapper.unmount()
  })

  it('isValid is false when task title is empty', () => {
    const { result, wrapper } = withSetup(() => useCapture())
    expect(result.isValid.value).toBe(false)
    wrapper.unmount()
  })

  it('isValid is true when task title is filled', () => {
    const { result, wrapper } = withSetup(() => useCapture())
    result.taskForm.value.title = 'My task'
    expect(result.isValid.value).toBe(true)
    wrapper.unmount()
  })

  it('setMode switches between task and context', () => {
    const { result, wrapper } = withSetup(() => useCapture())
    result.setMode('context')
    expect(result.mode.value).toBe('context')
    wrapper.unmount()
  })

  it('reset clears form fields', () => {
    const { result, wrapper } = withSetup(() => useCapture())
    result.taskForm.value.title = 'Something'
    result.reset()
    expect(result.taskForm.value.title).toBe('')
    wrapper.unmount()
  })

  it('submit creates task via taskService', async () => {
    const { taskService } = await import('@/services/taskService')
    const task = makeTask()
    vi.mocked(taskService.create).mockResolvedValue(task)

    const { result, wrapper } = withSetup(() => useCapture())
    result.taskForm.value.title = 'New task'

    await result.submit()
    await nextTick()

    expect(taskService.create).toHaveBeenCalled()

    wrapper.unmount()
  })
})
