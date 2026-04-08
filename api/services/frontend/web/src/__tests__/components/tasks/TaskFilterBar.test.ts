import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import TaskFilterBar from '@/components/tasks/TaskFilterBar.vue'

describe('TaskFilterBar', () => {
  it('renders status and priority selects', () => {
    const wrapper = mount(TaskFilterBar, { props: { filter: {} } })
    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(2)
  })

  it('emits update when status changes', async () => {
    const wrapper = mount(TaskFilterBar, { props: { filter: {} } })
    const statusSelect = wrapper.findAll('select')[0]!
    await statusSelect.setValue('open')
    await nextTick()
    const emitted = wrapper.emitted('update')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1]![0]).toMatchObject({ status: 'open' })
  })

  it('shows clear button when filter is active', async () => {
    const wrapper = mount(TaskFilterBar, { props: { filter: { status: 'open' } } })
    expect(wrapper.findAll('button').some(b => b.text() === 'Clear filters')).toBe(true)
  })

  it('emits empty filter on clear', async () => {
    const wrapper = mount(TaskFilterBar, { props: { filter: { status: 'open' } } })
    const clearButton = wrapper.findAll('button').find(b => b.text() === 'Clear filters')!
    await clearButton.trigger('click')
    const emitted = wrapper.emitted('update')
    expect(emitted![emitted!.length - 1]![0]).toEqual({ excludeStatuses: ['done', 'dismissed'] })
  })
})
