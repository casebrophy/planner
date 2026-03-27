import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ClarificationCard from '@/components/clarifications/ClarificationCard.vue'
import { makeClarificationItem } from '../../helpers/testFactories'
import { ClarificationKind } from '@/types'

describe('ClarificationCard', () => {
  it('renders question text', () => {
    const item = makeClarificationItem({ question: 'Is this task stale?' })
    const wrapper = mount(ClarificationCard, { props: { item } })
    expect(wrapper.text()).toContain('Is this task stale?')
  })

  it('renders kind label', () => {
    const item = makeClarificationItem({ kind: ClarificationKind.StaleTask })
    const wrapper = mount(ClarificationCard, { props: { item } })
    expect(wrapper.text()).toContain('Stale Task')
  })

  it('renders reasoning when present', () => {
    const item = makeClarificationItem({ reasoning: 'No activity for 2 weeks' })
    const wrapper = mount(ClarificationCard, { props: { item } })
    expect(wrapper.text()).toContain('No activity for 2 weeks')
  })

  it('shows stale task action buttons', () => {
    const item = makeClarificationItem({ kind: ClarificationKind.StaleTask })
    const wrapper = mount(ClarificationCard, { props: { item } })
    expect(wrapper.text()).toContain('Still active')
    expect(wrapper.text()).toContain('Add note')
    expect(wrapper.text()).toContain('Close')
  })

  it('emits resolve when action button clicked', async () => {
    const item = makeClarificationItem({ kind: ClarificationKind.StaleTask })
    const wrapper = mount(ClarificationCard, { props: { item } })
    const activeBtn = wrapper.findAll('button').find(b => b.text() === 'Still active')!
    await activeBtn.trigger('click')
    expect(wrapper.emitted('resolve')).toEqual([[{ action: 'extend' }]])
  })

  it('emits snooze with hours when snooze button clicked', async () => {
    const item = makeClarificationItem()
    const wrapper = mount(ClarificationCard, { props: { item } })
    const snoozeBtn = wrapper.findAll('button').find(b => b.text() === 'Snooze 24h')!
    await snoozeBtn.trigger('click')
    expect(wrapper.emitted('snooze')).toEqual([[24]])
  })

  it('emits dismiss when dismiss button clicked', async () => {
    const item = makeClarificationItem()
    const wrapper = mount(ClarificationCard, { props: { item } })
    const dismissBtn = wrapper.findAll('button').find(b => b.text() === 'Dismiss')!
    await dismissBtn.trigger('click')
    expect(wrapper.emitted('dismiss')).toHaveLength(1)
  })
})
