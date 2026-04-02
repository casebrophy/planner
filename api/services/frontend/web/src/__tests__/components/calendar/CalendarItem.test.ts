import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import CalendarItem from '@/components/calendar/CalendarItem.vue'
import type { ScheduleItem } from '@/types/schedule'

function makeItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    type: 'event',
    id: 'test-1',
    title: 'Test Event',
    startsAt: '2026-04-02T09:00:00Z',
    endsAt: '2026-04-02T10:00:00Z',
    ...overrides,
  }
}

describe('CalendarItem', () => {
  it('renders event title', () => {
    const wrapper = mount(CalendarItem, { props: { item: makeItem({ title: 'Meeting' }) } })
    expect(wrapper.text()).toContain('Meeting')
  })

  it('renders "Task block" when title is empty', () => {
    const wrapper = mount(CalendarItem, { props: { item: makeItem({ type: 'time_block', title: '' }) } })
    expect(wrapper.text()).toContain('Task block')
  })

  it('shows ? for unconfirmed time blocks', () => {
    const item = makeItem({ type: 'time_block', confirmed: false })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.text()).toContain('?')
  })

  it('does not show ? for confirmed time blocks', () => {
    const item = makeItem({ type: 'time_block', confirmed: true })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.find('[title="Unconfirmed"]').exists()).toBe(false)
  })

  it('does not show ? for events', () => {
    const item = makeItem({ type: 'event' })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.find('[title="Unconfirmed"]').exists()).toBe(false)
  })

  it('hides time for all-day items', () => {
    const item = makeItem({ allDay: true })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.findAll('.opacity-70')).toHaveLength(0)
  })

  it('shows time for timed items', () => {
    const item = makeItem({ allDay: false })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.find('.opacity-70').exists()).toBe(true)
  })

  it('emits click with item on click', async () => {
    const item = makeItem()
    const wrapper = mount(CalendarItem, { props: { item } })
    await wrapper.trigger('click')
    expect(wrapper.emitted('click')).toHaveLength(1)
    expect(wrapper.emitted('click')![0]![0]).toEqual(item)
  })

  it('applies blue styling for events', () => {
    const wrapper = mount(CalendarItem, { props: { item: makeItem({ type: 'event' }) } })
    expect(wrapper.find('.border-blue-500').exists()).toBe(true)
  })

  it('applies green styling for confirmed time blocks', () => {
    const item = makeItem({ type: 'time_block', confirmed: true })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.find('.border-green-500').exists()).toBe(true)
  })

  it('applies gray styling for unconfirmed time blocks', () => {
    const item = makeItem({ type: 'time_block', confirmed: false })
    const wrapper = mount(CalendarItem, { props: { item } })
    expect(wrapper.find('.border-gray-500').exists()).toBe(true)
  })
})
