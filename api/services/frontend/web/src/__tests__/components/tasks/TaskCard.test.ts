import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { makeTask } from '../../helpers/testFactories'

describe('TaskCard', () => {
  it('renders task title', () => {
    const task = makeTask({ title: 'Buy groceries' })
    const wrapper = mount(TaskCard, { props: { task } })
    expect(wrapper.text()).toContain('Buy groceries')
  })

  it('renders task description when present', () => {
    const task = makeTask({ description: 'Milk and bread' })
    const wrapper = mount(TaskCard, { props: { task } })
    expect(wrapper.text()).toContain('Milk and bread')
  })

  it('emits click with task id', async () => {
    const task = makeTask()
    const wrapper = mount(TaskCard, { props: { task } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toEqual([[task.id]])
  })

  it('shows overdue styling for past due tasks', () => {
    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const task = makeTask({ dueDate: pastDate, status: 'todo' })
    const wrapper = mount(TaskCard, { props: { task } })
    expect(wrapper.find('.text-red-400').exists()).toBe(true)
  })

  it('does not show overdue for completed tasks', () => {
    const pastDate = new Date(Date.now() - 86400000).toISOString()
    const task = makeTask({ dueDate: pastDate, status: 'done' })
    const wrapper = mount(TaskCard, { props: { task } })
    expect(wrapper.find('.text-red-400').exists()).toBe(false)
  })
})
