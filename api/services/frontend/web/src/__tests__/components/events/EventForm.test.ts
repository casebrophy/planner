import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EventForm from '@/components/events/EventForm.vue'

describe('EventForm', () => {
  it('renders kind select and content textarea', () => {
    const wrapper = mount(EventForm)
    expect(wrapper.find('select').exists()).toBe(true)
    expect(wrapper.find('textarea').exists()).toBe(true)
  })

  it('disables submit when content is empty', () => {
    const wrapper = mount(EventForm)
    const submitBtn = wrapper.find('button[type="submit"]')
    expect(submitBtn.attributes('disabled')).toBeDefined()
  })

  it('emits submit with NewEvent data', async () => {
    const wrapper = mount(EventForm)
    await wrapper.find('textarea').setValue('Something happened')
    await wrapper.find('form').trigger('submit')
    const emitted = wrapper.emitted('submit')
    expect(emitted).toHaveLength(1)
    expect(emitted![0]![0]).toMatchObject({ kind: 'note', content: 'Something happened' })
  })

  it('clears content after submit', async () => {
    const wrapper = mount(EventForm)
    await wrapper.find('textarea').setValue('Something happened')
    await wrapper.find('form').trigger('submit')
    expect((wrapper.find('textarea').element as HTMLTextAreaElement).value).toBe('')
  })
})
