import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'
import ContextFilterBar from '@/components/contexts/ContextFilterBar.vue'

describe('ContextFilterBar', () => {
  it('renders search input and status select', () => {
    const wrapper = mount(ContextFilterBar, { props: { filter: {} } })
    expect(wrapper.find('input').exists()).toBe(true)
    expect(wrapper.find('select').exists()).toBe(true)
  })

  it('emits update when status changes', async () => {
    const wrapper = mount(ContextFilterBar, { props: { filter: {} } })
    await wrapper.find('select').setValue('active')
    await nextTick()
    const emitted = wrapper.emitted('update')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1]![0]).toMatchObject({ status: 'active' })
  })

  it('emits update when title search changes', async () => {
    const wrapper = mount(ContextFilterBar, { props: { filter: {} } })
    await wrapper.find('input').setValue('home')
    await nextTick()
    const emitted = wrapper.emitted('update')
    expect(emitted).toBeTruthy()
    expect(emitted![emitted!.length - 1]![0]).toMatchObject({ title: 'home' })
  })

  it('shows clear button when filter active', () => {
    const wrapper = mount(ContextFilterBar, { props: { filter: { status: 'active' } } })
    expect(wrapper.find('button').exists()).toBe(true)
  })
})
