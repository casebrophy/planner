import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import EventTimeline from '@/components/events/EventTimeline.vue'
import { makeContextEvent } from '../../helpers/testFactories'

describe('EventTimeline', () => {
  it('renders events', () => {
    const events = [makeContextEvent({ content: 'First event' }), makeContextEvent({ content: 'Second event' })]
    const wrapper = mount(EventTimeline, { props: { events } })
    expect(wrapper.text()).toContain('First event')
    expect(wrapper.text()).toContain('Second event')
  })

  it('shows empty message when no events', () => {
    const wrapper = mount(EventTimeline, { props: { events: [] } })
    expect(wrapper.text()).toContain('No events yet')
  })
})
