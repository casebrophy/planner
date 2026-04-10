# Schedule System

> Schedule provides a unified read-only view merging TimeBlocks and CalendarEvents for a given date range. It powers the calendar UI, daily views, and week grids by combining two separate entity types into a single sorted stream. The service queries /api/v1/schedule with start/end timestamps and returns a flat list of schedulable items.

## Core Types

```typescript
export interface ScheduleItem {
  type: 'event' | 'time_block'
  id: string
  title: string
  startsAt: string
  endsAt: string
  allDay?: boolean
  location?: string
  taskId?: string
  confirmed?: boolean
}

export interface ScheduleResponse {
  items: ScheduleItem[]
}
```

## File Map

### Services
- `services/scheduleService.ts` — **scheduleService** — read-only service fetching /api/v1/schedule for a date range, returns merged ScheduleResponse

### Composables
- `composables/useCalendar.ts` — **useCalendar** — fetches schedule via scheduleService, separates items back into TimeBlocks and CalendarEvents for display

### Components
- `components/calendar/WeekGrid.vue` — renders items from useCalendar (filtered schedule items)
- `components/calendar/CalendarItem.vue` — displays single ScheduleItem with type-specific rendering (event vs time block)
- `components/tasks/TaskCard.vue` — may render upcoming time blocks if task has scheduledAt

## Impact Callouts

### ⚠ ScheduleItem (types/schedule.ts)
Changing this interface affects:
- `services/scheduleService.ts` — response deserialization from /api/v1/schedule
- `composables/useCalendar.ts` — iterates items array, filters by date range, separates by type
- `components/calendar/WeekGrid.vue` — maps ScheduleItem array to grid cells
- `components/calendar/CalendarItem.vue` — template binds type discriminator, title, startsAt, endAt, location, confirmed

### ⚠ ScheduleResponse (types/schedule.ts)
Changing this interface affects:
- `services/scheduleService.ts` — wraps response from API
- `composables/useCalendar.ts` — destructures .items array from response

## Cross-Domain Dependencies

- **timeBlock domain** — ScheduleItem with type='time_block' maps to TimeBlock fields
- **calendarEvent domain** — ScheduleItem with type='event' maps to CalendarEvent fields
- **calendar views** — WeekGrid, CalendarView, ContextDetailView all depend on scheduleService for unified timeline
