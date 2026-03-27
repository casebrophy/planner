import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia } from 'pinia'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { makeTask } from '../../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/contextService', () => ({
  contextService: { list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 50 }) },
}))

describe('TaskForm', () => {
  function mountForm(props = {}) {
    return mount(TaskForm, {
      props: { mode: 'create' as const, ...props },
      global: { plugins: [createPinia()] },
    })
  }

  it('renders create mode with empty fields', () => {
    const wrapper = mountForm()
    expect(wrapper.find('button[type="submit"]').text()).toBe('Create Task')
  })

  it('renders edit mode with populated fields', () => {
    const task = makeTask({ title: 'Existing Task' })
    const wrapper = mountForm({ mode: 'edit', task })
    expect(wrapper.find('button[type="submit"]').text()).toBe('Save Changes')
    const titleInput = wrapper.find('input[type="text"]')
    expect((titleInput.element as HTMLInputElement).value).toBe('Existing Task')
  })

  it('disables submit when title is empty', () => {
    const wrapper = mountForm()
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeDefined()
  })

  it('emits submit with NewTask data in create mode', async () => {
    const wrapper = mountForm()
    await wrapper.find('input[type="text"]').setValue('New Task')
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toMatchObject({ title: 'New Task' })
  })

  it('emits cancel when cancel button clicked', async () => {
    const wrapper = mountForm()
    const cancelBtn = wrapper.findAll('button').find(b => b.text() === 'Cancel')!
    await cancelBtn.trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('shows status field only in edit mode', () => {
    const createWrapper = mountForm()
    const editWrapper = mountForm({ mode: 'edit', task: makeTask() })
    expect(createWrapper.findAll('select').length).toBeLessThan(editWrapper.findAll('select').length)
  })
})
