import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import CalendarEventForm from '@/components/calendar-events/CalendarEventForm.vue'
import type { NewCalendarEvent } from '@/types/calendarEvent'

vi.mock('@/stores/contextStore', () => ({
  useContextStore: () => ({
    items: [],
    fetchList: vi.fn(),
    fetchAll: vi.fn(),
  }),
}))

beforeEach(() => {
  setActivePinia(createPinia())
})

describe('CalendarEventForm — initialContextId', () => {
  it('hides context picker when initialContextId is provided', () => {
    const wrapper = mount(CalendarEventForm, {
      props: { initialContextId: 'ctx-123' },
    })
    const labels = wrapper.findAll('label')
    const contextLabel = labels.find((l) => l.text() === 'Context')
    expect(contextLabel).toBeUndefined()
  })

  it('shows context picker when initialContextId is not provided', () => {
    const wrapper = mount(CalendarEventForm, {
      props: {},
    })
    const labels = wrapper.findAll('label')
    const contextLabel = labels.find((l) => l.text() === 'Context')
    expect(contextLabel).toBeDefined()
  })

  it('pre-populates contextId in emitted save payload when initialContextId is set', async () => {
    const wrapper = mount(CalendarEventForm, {
      props: { initialContextId: 'ctx-123' },
    })
    await wrapper.find('input[type="text"]').setValue('Team standup')
    const datetimeInputs = wrapper.findAll('input[type="datetime-local"]')
    await datetimeInputs[0]!.setValue('2026-04-10T10:00')
    await datetimeInputs[1]!.setValue('2026-04-10T11:00')
    await wrapper.find('form').trigger('submit')

    const emitted = wrapper.emitted('save')
    expect(emitted).toHaveLength(1)
    expect((emitted![0]![0] as NewCalendarEvent).contextId).toBe('ctx-123')
  })
})
