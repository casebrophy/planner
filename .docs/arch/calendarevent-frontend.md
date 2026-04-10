# CalendarEvent System

> CalendarEvents represent scheduled moments (meetings, deadlines, milestones) optionally tied to contexts and time windows. They provide a lightweight calendar layer separate from tasks, with support for all-day events, locations, and optional raw input traceability. CalendarEvents integrate with contexts for organizational scope and merge with TimeBlocks in the unified schedule.

## Core Types

```typescript
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
```

## File Map

### Stores
- `stores/calendarEventStore.ts` — **useCalendarEventStore** — CRUD store wrapping calendarEventService, manages pagination by date range

### Services
- `services/calendarEventService.ts` — **calendarEventService** — CRUD HTTP client for /api/v1/events endpoint

### Composables
- `composables/useEventBoard.ts` — **useEventBoard** — manages event list state, filtering by context, pagination, create/edit dialogs

### Components
- `components/calendar-events/CalendarEventCard.vue` — **CalendarEventCard** — displays single event with title, time, location, context badge
- `components/calendar-events/CalendarEventForm.vue` — **CalendarEventForm** — form for creating/editing events with context picker

## Impact Callouts

### ⚠ CalendarEvent (types/calendarEvent.ts)
Changing this interface shape affects:
- `stores/calendarEventStore.ts` — CRUD operations normalize/denormalize via createCRUDStore
- `services/calendarEventService.ts` — request/response serialization (mapFilter translates dateFrom/dateTo/contextId to query params)
- `composables/useEventBoard.ts` — filters by contextId and date range, manages selected event state
- `components/calendar-events/CalendarEventCard.vue` — template binds title, description, location, startsAt, allDay, context relationship
- `components/calendar-events/CalendarEventForm.vue` — form fields for creation/update
- `views/EventsView.vue` — lists all calendar events with filtering by context
- `views/ContextDetailView.vue` — displays context's calendar events in embedded list

## Cross-Domain Dependencies

- **context domain** — CalendarEvent.contextId references Context (optional), linked for organizational grouping
- **schedule service** — calendarEventService events merge with TimeBlocks via scheduleService.getSchedule()
- **rawinput domain** — CalendarEvent.rawInputId optionally traces event back to ingested content
