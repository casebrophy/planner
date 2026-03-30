import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EnergyIndicator from '@/components/shared/EnergyIndicator.vue'

describe('EnergyIndicator', () => {
  it('renders energy label', () => {
    const wrapper = mount(EnergyIndicator, { props: { energy: 'low' } })
    expect(wrapper.text()).toBe('Low')
  })

  it('renders 1 active bar for low energy', () => {
    const wrapper = mount(EnergyIndicator, { props: { energy: 'low' } })
    const bars = wrapper.findAll('.flex.gap-0\\.5 span')
    const activeBars = bars.filter(b => b.classes().includes('bg-amber-500'))
    expect(activeBars).toHaveLength(1)
  })

  it('renders 3 active bars for high energy', () => {
    const wrapper = mount(EnergyIndicator, { props: { energy: 'high' } })
    const bars = wrapper.findAll('.flex.gap-0\\.5 span')
    const activeBars = bars.filter(b => b.classes().includes('bg-amber-500'))
    expect(activeBars).toHaveLength(3)
  })
})
