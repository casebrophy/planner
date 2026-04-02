import { createCRUDService } from './createCRUDService'
import type { CalendarEvent, NewCalendarEvent, UpdateCalendarEvent, CalendarEventFilter } from '@/types/calendarEvent'

export const calendarEventService = createCRUDService<CalendarEvent, NewCalendarEvent, UpdateCalendarEvent, CalendarEventFilter>({
  basePath: '/api/v1/events',
  mapFilter: (f) => ({
    date_from: f.dateFrom,
    date_to: f.dateTo,
    context_id: f.contextId,
  }),
})
