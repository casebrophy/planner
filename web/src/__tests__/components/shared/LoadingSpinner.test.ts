import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'

describe('LoadingSpinner', () => {
  it('renders with default medium size', () => {
    const wrapper = mount(LoadingSpinner)
    const svg = wrapper.find('svg')
    expect(svg.classes()).toContain('w-8')
    expect(svg.classes()).toContain('h-8')
  })

  it('renders with small size', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'sm' } })
    const svg = wrapper.find('svg')
    expect(svg.classes()).toContain('w-5')
  })

  it('renders with large size', () => {
    const wrapper = mount(LoadingSpinner, { props: { size: 'lg' } })
    const svg = wrapper.find('svg')
    expect(svg.classes()).toContain('w-12')
  })
})
