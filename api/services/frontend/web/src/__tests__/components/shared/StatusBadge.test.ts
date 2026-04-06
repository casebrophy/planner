import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import StatusBadge from '@/components/shared/StatusBadge.vue'

describe('StatusBadge', () => {
  it('renders task status label', () => {
    const wrapper = mount(StatusBadge, { props: { status: 'open', type: 'task' } })
    expect(wrapper.text()).toBe('Open')
  })

  it('renders context status label', () => {
    const wrapper = mount(StatusBadge, { props: { status: 'active', type: 'context' } })
    expect(wrapper.text()).toBe('Active')
  })

  it('falls back to raw status when type not provided', () => {
    const wrapper = mount(StatusBadge, { props: { status: 'blocked' } })
    expect(wrapper.text()).toBe('Blocked')
  })

  it('applies correct color style', () => {
    const wrapper = mount(StatusBadge, { props: { status: 'done', type: 'task' } })
    const span = wrapper.find('span')
    expect(span.attributes('style')).toContain('#22c55e')
  })
})
