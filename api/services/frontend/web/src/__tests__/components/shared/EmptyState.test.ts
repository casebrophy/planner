import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EmptyState from '@/components/shared/EmptyState.vue'

describe('EmptyState', () => {
  it('renders title', () => {
    const wrapper = mount(EmptyState, { props: { title: 'No items' } })
    expect(wrapper.text()).toContain('No items')
  })

  it('renders optional message', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Empty', message: 'Nothing here' } })
    expect(wrapper.text()).toContain('Nothing here')
  })

  it('shows action button when actionLabel provided', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Empty', actionLabel: 'Create' } })
    expect(wrapper.find('button').text()).toBe('Create')
  })

  it('emits action when button clicked', async () => {
    const wrapper = mount(EmptyState, { props: { title: 'Empty', actionLabel: 'Create' } })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('action')).toHaveLength(1)
  })

  it('hides action button when no actionLabel', () => {
    const wrapper = mount(EmptyState, { props: { title: 'Empty' } })
    expect(wrapper.find('button').exists()).toBe(false)
  })
})
