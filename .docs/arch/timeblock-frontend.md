# TimeBlock System

> TimeBlocks schedule tasks onto the calendar, mapping task IDs to specific time windows. They integrate with the task domain to provide calendar-based scheduling and conflict detection. The system supports confirmation state, enabling users to tentatively block time before committing.

## Core Types

```typescript
export interface TimeBlock {
  id: string
  taskId: string
  startsAt: string
  endsAt: string
  confirmed: boolean
  createdAt: string
  updatedAt: string
}

export interface NewTimeBlock {
  taskId: string
  startsAt: string
  endsAt: string
}

export interface UpdateTimeBlock {
  startsAt?: string
  endsAt?: string
  confirmed?: boolean
}

export interface TimeBlockFilter {
  taskId?: string
  dateFrom?: string
  dateTo?: string
  confirmed?: string
}
```

## File Map

### Stores
- `stores/timeBlockStore.ts` — **useTimeBlockStore** — CRUD store wrapping timeBlockService, manages pagination, filtering, and selection state

### Services
- `services/timeBlockService.ts` — **timeBlockService** — CRUD HTTP client for /api/v1/time-blocks endpoint

### Composables
- `composables/useCalendar.ts` — **useCalendar** — orchestrates calendar UI state, fetches schedule and time blocks, manages day/week views

### Components
- `components/calendar/TimeBlockForm.vue` — **TimeBlockForm** — form for creating/editing time blocks, integrates with task picker

## Impact Callouts

### ⚠ TimeBlock (types/timeBlock.ts)
Changing this interface shape affects:
- `stores/timeBlockStore.ts` — CRUD operations normalize/denormalize via createCRUDStore
- `services/timeBlockService.ts` — request/response serialization (mapFilter translates UI filters to API query params)
- `composables/useCalendar.ts` — filters by dateFrom/dateTo, reads confirmed status
- `components/calendar/TimeBlockForm.vue` — template binds startsAt, endsAt, confirmed fields
- `views/CalendarView.vue` — renders time blocks alongside calendar events in WeekGrid

## Cross-Domain Dependencies

- **task domain** — TimeBlock.taskId references Task, calendar views fetch both together
- **schedule service** — useCalendar merges TimeBlocks and CalendarEvents via scheduleService.getSchedule()
- **enums** — no direct enum usage (confirmed is boolean)
