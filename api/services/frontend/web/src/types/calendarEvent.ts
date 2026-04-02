export interface CalendarEvent {
  id: string
  contextId?: string
  title: string
  description: string
  location?: string
  startsAt: string
  endsAt: string
  allDay: boolean
  rawInputId?: string
  createdAt: string
  updatedAt: string
}

export interface NewCalendarEvent {
  contextId?: string
  title: string
  description: string
  location?: string
  startsAt: string
  endsAt: string
  allDay: boolean
}

export interface UpdateCalendarEvent {
  contextId?: string
  title?: string
  description?: string
  location?: string
  startsAt?: string
  endsAt?: string
  allDay?: boolean
}

export interface CalendarEventFilter {
  dateFrom?: string
  dateTo?: string
  contextId?: string
}
