import { defineStore } from 'pinia'
import { createCRUDStore } from './createCRUDStore'
import { calendarEventService } from '@/services/calendarEventService'
import type { CalendarEvent, NewCalendarEvent, UpdateCalendarEvent, CalendarEventFilter } from '@/types/calendarEvent'

export const useCalendarEventStore = defineStore('calendarEvent', () => {
  const crud = createCRUDStore<CalendarEvent, NewCalendarEvent, UpdateCalendarEvent, CalendarEventFilter>({
    name: 'calendarEvent',
    service: calendarEventService,
    defaultOrderBy: 'starts_at',
    defaultRowsPerPage: 20,
  })

  return { ...crud }
})
