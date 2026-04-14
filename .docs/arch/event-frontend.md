# Event System

> ContextEvents are lightweight activity log entries tied to a context, recording significant moments, actions, or milestones with optional metadata. They are displayed in the daily plan and context timelines, providing a simple event/activity record without time windows (unlike CalendarEvents). ContextEvents support arbitrary kind/content pairs and optional source traceability.

## Core Types

```typescript
export interface ContextEvent {
  id: string
  contextId: string
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
  createdAt: string
}

export interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
```

## File Map

### Components
- `components/dailyplan/EventCard.vue` — **EventCard** — displays a single ContextEvent with kind and content in a styled card; designed for timeline/activity list display

### Test Helpers
- `__tests__/helpers/testFactories.ts` — **makeContextEvent()** — factory function for creating test ContextEvent instances

## Impact Callouts

### ⚠ ContextEvent (types/event.ts)
Changing this interface shape affects:
- `components/dailyplan/EventCard.vue` — template reads event.kind and event.content for display; metadata and sourceId available for future expansion
- `__tests__/helpers/testFactories.ts` — test factory must generate all required fields (id, contextId, kind, content, createdAt)
- Any future store/service that manages ContextEvents (not yet implemented)

## Cross-Domain Dependencies

- **dailyplan domain** — EventCard component is used by DailyPlanView to display context events in the activity stream
- **context domain** — ContextEvent.contextId references Context (required), events are scoped to a specific context
- **rawinput domain** — ContextEvent.sourceId optionally traces event back to ingested content source
