import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useCalendar } from '@/composables/useCalendar'
import { scheduleService } from '@/services/scheduleService'
import type { ScheduleItem } from '@/types/schedule'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/scheduleService', () => ({
  scheduleService: {
    getSchedule: vi.fn(),
  },
}))

vi.mock('@/services/timeBlockService', () => ({
  timeBlockService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

function withSetup<T>(composable: () => T) {
  let result!: T
  const wrapper = mount(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
    { global: { plugins: [createPinia()] } },
  )
  return { result, wrapper }
}

function makeScheduleItem(overrides: Partial<ScheduleItem> = {}): ScheduleItem {
  return {
    type: 'event',
    id: `item-${Math.random()}`,
    title: 'Test Item',
    startsAt: new Date().toISOString(),
    endsAt: new Date(Date.now() + 3600000).toISOString(),
    ...overrides,
  }
}

describe('useCalendar', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('weekDays returns 7 days', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    expect(result.weekDays.value).toHaveLength(7)
    wrapper.unmount()
  })

  it('fetches schedule on mount', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    expect(scheduleService.getSchedule).toHaveBeenCalledTimes(1)
    wrapper.unmount()
  })

  it('separates allDay and timed items', async () => {
    const allDayEvent = makeScheduleItem({ allDay: true, title: 'All Day' })
    const timedEvent = makeScheduleItem({ allDay: false, title: 'Timed' })
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [allDayEvent, timedEvent] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    expect(result.allDayItems.value).toHaveLength(1)
    expect(result.allDayItems.value[0]!.title).toBe('All Day')
    expect(result.timedItems.value).toHaveLength(1)
    expect(result.timedItems.value[0]!.title).toBe('Timed')
    wrapper.unmount()
  })

  it('nextWeek advances the current date by one week', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    const initialDate = result.currentDate.value.getTime()
    result.nextWeek()
    await nextTick()
    await nextTick()

    const newDate = result.currentDate.value.getTime()
    const diffDays = (newDate - initialDate) / (1000 * 60 * 60 * 24)
    expect(diffDays).toBe(7)
    wrapper.unmount()
  })

  it('prevWeek goes back one week', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    const initialDate = result.currentDate.value.getTime()
    result.prevWeek()
    await nextTick()
    await nextTick()

    const newDate = result.currentDate.value.getTime()
    const diffDays = (initialDate - newDate) / (1000 * 60 * 60 * 24)
    expect(diffDays).toBe(7)
    wrapper.unmount()
  })

  it('openCreateAtSlot sets form state', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()

    const start = '2026-04-02T09:00:00.000Z'
    const end = '2026-04-02T10:00:00.000Z'
    result.openCreateAtSlot(start, end)

    expect(result.showForm.value).toBe(true)
    expect(result.formStartsAt.value).toBe(start)
    expect(result.formEndsAt.value).toBe(end)
    wrapper.unmount()
  })

  it('closeForm resets form state', async () => {
    vi.mocked(scheduleService.getSchedule).mockResolvedValue({ items: [] })

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()

    result.openCreateAtSlot('2026-04-02T09:00:00Z', '2026-04-02T10:00:00Z')
    result.closeForm()

    expect(result.showForm.value).toBe(false)
    expect(result.formStartsAt.value).toBe('')
    expect(result.formEndsAt.value).toBe('')
    wrapper.unmount()
  })

  it('handles fetch error gracefully', async () => {
    vi.mocked(scheduleService.getSchedule).mockRejectedValue(new Error('Network error'))

    const { result, wrapper } = withSetup(() => useCalendar())
    await nextTick()
    await nextTick()

    expect(result.items.value).toEqual([])
    expect(result.loading.value).toBe(false)
    wrapper.unmount()
  })
})
