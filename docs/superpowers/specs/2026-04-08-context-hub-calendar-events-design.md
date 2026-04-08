# Context Hub: Calendar Events Cleanup & Surface

**Date:** 2026-04-08  
**Branch:** fix/2026-04-08

## Problem

The context hub (ContextDetailView) has two issues:

1. The "Events" collapsible shows `context_events` — raw text log entries added via voice/ingest. These are not scheduled events. The label is misleading.
2. Actual calendar events (`events` table) are shown in the project hub Timeline but nowhere in the area hub, and there's no way to create a calendar event linked to a context from within the hub.
3. The thread domain (`thread_entries`) is the designed successor to `context_events` and is already surfaced via `ThreadPanel`. Both systems are rendered, creating duplication. `context_events` is legacy.

## Decision

- Remove the "Events" collapsible (context_events) from both project and area hubs.
- Add a sidebar "Events" card in both hubs showing actual calendar events, with inline create via `CalendarEventForm`.
- Keep the `ThreadPanel` collapsible in both hubs — it's the correct long-term activity log system.

## What Gets Removed

- `EventTimeline.vue`, `EventTimelineItem.vue`, `EventForm.vue` — only used in `ContextDetailView`; can be deleted
- `events`, `eventsTotal`, `addEvent` from `useContextDetail` composable
- `contextStore.fetchEvents()` call and related store refs
- The "Events" collapsible block from both project and area hub templates
- Unused imports: `NewEvent`, `EventTimeline`, `EventForm`

## What Gets Added

### `CalendarEventForm` — new `initialContextId` prop

Add an optional `initialContextId: string` prop. When provided:
- Pre-populates `contextId` ref on mount
- Hides the context picker `<select>` (user is already in a context)

### Sidebar Events card (both hubs)

Position:
- **Project hub sidebar**: after Tags, before Observations
- **Area hub sidebar**: after Tags, before Notes

Contents:
- Header "Events" + small "+ Add" button
- List of `contextCalendarEvents` via existing `CalendarEventCard` (handles past opacity, location badge, delete)
- Empty state: "No events yet"
- "+ Add" opens a `DrawerPanel` with `CalendarEventForm` (initialContextId pre-set)

### ContextDetailView changes

- New `showAddEvent` ref controls the create drawer
- `handleCreateCalendarEvent(data)`: calls `calendarEventStore.create(data)`, then `calendarEventStore.fetchList(true)`
- `handleDeleteCalendarEvent` already exists — reuse in sidebar card

## Data Flow

`contextCalendarEvents` is already fetched on mount via `calendarEventStore.setFilter({ contextId })` + `fetchList(true)`. No new fetch needed.

Create: `calendarEventStore.create(data)` → `calendarEventStore.fetchList(true)`  
Delete: existing `handleDeleteCalendarEvent(id)` reused

**No backend changes required.** `calendarEventService` already supports `contextId` on create/update.

## Files Changed

| File | Action |
|------|--------|
| `src/components/events/EventForm.vue` | Delete |
| `src/components/events/EventTimeline.vue` | Delete |
| `src/components/events/EventTimelineItem.vue` | Delete |
| `src/composables/useContextDetail.ts` | Remove events/eventsTotal/addEvent |
| `src/stores/contextStore.ts` | Remove fetchEvents, addEvent, events, eventsTotal |
| `src/services/contextService.ts` | Remove addEvent method |
| `src/components/calendar-events/CalendarEventForm.vue` | Add `initialContextId` prop |
| `src/views/ContextDetailView.vue` | Remove Events collapsible, add sidebar Events card + drawer |
| `src/__tests__/stores/contextStore.test.ts` | Remove fetchEvents/addEvent tests |
| `src/__tests__/services/contextService.test.ts` | Remove addEvent test |
| `src/__tests__/views/ContextDetailView.test.ts` | Update mock shape |
| `src/__tests__/components/events/EventForm.test.ts` | Delete |
| `src/__tests__/components/events/EventTimeline.test.ts` | Delete |
| All other test files with `addEvent: vi.fn()` in contextStore mock | Remove addEvent from mock shape |

## Out of Scope

- Backend changes
- Migrating or deleting `context_events` data
- Wiring thread entry writes (separate concern)
- Edit calendar events from context hub (add only for now)
