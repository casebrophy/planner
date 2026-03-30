import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ContextKanban from '@/components/contexts/ContextKanban.vue'
import { makeContext } from '../../helpers/testFactories'

describe('ContextKanban', () => {
  const columns = {
    active: [makeContext({ status: 'active' }), makeContext({ status: 'active' })],
    paused: [makeContext({ status: 'paused' })],
    closed: [],
  }

  it('renders three columns', () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    expect(wrapper.text()).toContain('Active')
    expect(wrapper.text()).toContain('Paused')
    expect(wrapper.text()).toContain('Closed')
  })

  it('renders correct count per column', () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    expect(wrapper.text()).toContain('(2)')
    expect(wrapper.text()).toContain('(1)')
    expect(wrapper.text()).toContain('(0)')
  })

  it('shows "No contexts" in empty columns', () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    expect(wrapper.text()).toContain('No contexts')
  })

  it('emits select when a context card is clicked', async () => {
    const wrapper = mount(ContextKanban, { props: { columns } })
    const firstCard = wrapper.findComponent({ name: 'ContextCard' })
    await firstCard.trigger('click')
    expect(wrapper.emitted('select')).toBeTruthy()
  })
})
