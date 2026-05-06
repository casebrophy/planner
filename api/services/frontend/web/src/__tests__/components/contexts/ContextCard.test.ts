import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ContextCard from '@/components/contexts/ContextCard.vue'
import { makeContext } from '../../helpers/testFactories'
import { ContextKind } from '@/types'

describe('ContextCard', () => {
  it('renders context title', () => {
    const ctx = makeContext({ title: 'Home Renovation' })
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('Home Renovation')
  })

  it('renders description when present', () => {
    const ctx = makeContext({ description: 'Kitchen and bath' })
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('Kitchen and bath')
  })

  it('emits click with context id', async () => {
    const ctx = makeContext()
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toEqual([[ctx.id]])
  })

  it('shows last event label when lastEvent is set', () => {
    const ctx = makeContext({ lastEvent: new Date().toISOString() })
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('Last event')
  })

  it('shows "No events yet" when no lastEvent', () => {
    const ctx = makeContext()
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('No events yet')
  })

  it('renders Project badge for project context', () => {
    const ctx = makeContext({ kind: ContextKind.Project })
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('Project')
  })

  it('renders Area badge for area context', () => {
    const ctx = makeContext({ kind: ContextKind.Area })
    const wrapper = mount(ContextCard, { props: { context: ctx } })
    expect(wrapper.text()).toContain('Area')
  })
})
