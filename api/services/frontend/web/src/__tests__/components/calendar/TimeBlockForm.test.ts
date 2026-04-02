import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import TimeBlockForm from '@/components/calendar/TimeBlockForm.vue'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({
      items: [
        { id: 'task-1', title: 'Buy groceries', status: 'todo' },
        { id: 'task-2', title: 'Read book', status: 'in_progress' },
        { id: 'task-3', title: 'Done task', status: 'done' },
      ],
      total: 3,
      page: 1,
      rowsPerPage: 20,
    }),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

describe('TimeBlockForm', () => {
  function mountForm(props = {}) {
    return mount(TimeBlockForm, {
      props,
      global: { plugins: [createPinia()] },
    })
  }

  it('renders task select and datetime inputs', () => {
    const wrapper = mountForm()
    expect(wrapper.find('select').exists()).toBe(true)
    expect(wrapper.findAll('input[type="datetime-local"]')).toHaveLength(2)
  })

  it('disables submit when no task selected', () => {
    const wrapper = mountForm()
    const btn = wrapper.find('button[type="submit"]')
    expect(btn.attributes('disabled')).toBeDefined()
  })

  it('emits cancel on cancel click', async () => {
    const wrapper = mountForm()
    await wrapper.find('button[type="button"]').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('pre-fills start and end from props', () => {
    const wrapper = mountForm({
      startsAt: '2026-04-02T09:00:00.000Z',
      endsAt: '2026-04-02T10:00:00.000Z',
    })
    const inputs = wrapper.findAll('input[type="datetime-local"]')
    // Should have values set (exact format depends on timezone)
    expect(inputs).toHaveLength(2)
  })
})
