import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import PriorityIndicator from '@/components/shared/PriorityIndicator.vue'

describe('PriorityIndicator', () => {
  it('renders priority label', () => {
    const wrapper = mount(PriorityIndicator, { props: { priority: 'high' } })
    expect(wrapper.text()).toBe('High')
  })

  it('applies correct color for urgent priority', () => {
    const wrapper = mount(PriorityIndicator, { props: { priority: 'urgent' } })
    const dot = wrapper.find('span span')
    expect(dot.attributes('style')).toContain('#ef4444')
  })
})
