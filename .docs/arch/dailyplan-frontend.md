# Daily Plan System (Frontend)

> AI-generated daily plan with grouped, prioritized tasks. Users can complete, dismiss (with reason), and drag-to-reorder items. Plan data is fetched from the backend, stored in a Pinia store, and composed via `useDailyPlan`. Dismiss reasons are tracked for retrospective analysis. Polling keeps the view fresh without manual refresh.

## Core Types

```typescript
// src/types/dailyPlan.ts

export interface DailyPlan {
  id: string
  planDate: string
  generation: number
  modelUsed: string
  createdAt: string
  items: DailyPlanItem[]
}

export interface DailyPlanItem {
  id: string
  planId: string
  taskId: string
  position: number          // AI-assigned position
  groupName: string         // e.g. "Deep Work", "Quick Wins"
  groupPosition: number     // Group sort order
  aiDurationMin?: number    // AI-suggested duration
  aiPriorityReason?: string // AI rationale shown in card
  userPosition?: number     // User drag override
  userDurationMin?: number  // User duration override
  status: string            // 'pending' | 'completed' | 'dismissed'
  dismissReason?: string    // completed_elsewhere | not_relevant | postpone | blocked | other
  dismissNote?: string
  completedAt?: string
  createdAt: string
}

export interface UpdatePlanItem {
  userPosition?: number
  userDurationMin?: number
}

export interface DismissRequest {
  reason: string
  note?: string
}

// From src/services/dailyPlanService.ts (local, not in types/)
export interface GenerateAccepted {
  status: string
}
```

Re-exported via `src/types/index.ts`:
```typescript
export type { DailyPlan, DailyPlanItem, UpdatePlanItem, DismissRequest } from './dailyPlan'
```

## File Map

### Stores
- `stores/dailyPlanStore.ts` — **useDailyPlanStore** — Pinia store; holds `plan`, `loading`, `generating`; all mutations optimistic with rollback on failure

### Services
- `services/dailyPlanService.ts` — **dailyPlanService** — API client; `getPlan`, `generate`, `updateItem`, `completeItem`, `dismissItem`

### Composables
- `composables/useDailyPlan.ts` — **useDailyPlan** — View-layer composition; groups items by `groupName`, builds `taskMap`, computes stats, mounts polling via `usePolling`

### Views
- `views/DailyPlanView.vue` — **DailyPlanView** — Full-page view; orchestrates groups, drag-and-drop, inline dismiss modal; route `/plan`

### Components
- `components/dailyplan/PlanItemCard.vue` — **PlanItemCard** — Renders one plan item with drag handle, priority color bar (from `Task.priority`), AI reason, duration badge, complete/dismiss buttons
- `components/dailyplan/PlanGroupHeader.vue` — **PlanGroupHeader** — Section header showing group name (uppercase) and item count badge

## Impact Callouts

### ⚠ DailyPlan (types/dailyPlan.ts)
Changing this interface shape affects:
- `stores/dailyPlanStore.ts` — `plan` ref typed as `DailyPlan | null`; `fetchPlan` sets it directly from API response
- `composables/useDailyPlan.ts` — `plan.value?.items` iterated for grouping, completion stats, task map; `plan.value` passed to view
- `views/DailyPlanView.vue` — `plan.value?.items`, `hasPlan` computed, subtitle completion ratio

### ⚠ DailyPlanItem (types/dailyPlan.ts)
Changing this interface shape affects:
- `stores/dailyPlanStore.ts` — `completeItem` mutates `item.status`, `item.completedAt`; `dismissItem` mutates `item.status`, `item.dismissReason`, `item.dismissNote`; `reorderItem` mutates `item.userPosition`
- `composables/useDailyPlan.ts` — groups filtered by `item.status !== 'dismissed'`; sorted by `item.userPosition ?? item.position` and `item.groupPosition`; `completedCount`/`totalCount` filter by `item.status`
- `views/DailyPlanView.vue` — `handleReorder` reads `item.userPosition`, `item.position`, `item.id`; `openTask` reads `item.taskId`; dismiss modal writes `dismissReason`/`dismissNote`
- `components/dailyplan/PlanItemCard.vue` — renders `item.status` (strikethrough, hide actions), `item.aiPriorityReason`, `item.userDurationMin ?? item.aiDurationMin`; emits `item.id` on actions
- `services/dailyPlanService.ts` — `updateItem`/`completeItem`/`dismissItem` return `DailyPlanItem` from API

### ⚠ DismissRequest (types/dailyPlan.ts)
Changing this interface shape affects:
- `stores/dailyPlanStore.ts` — `dismissItem(itemId, reason, note?)` constructs `{ reason, note }` and passes to service
- `services/dailyPlanService.ts` — `dismissItem(itemId, data: DismissRequest)` sends as POST body
- `views/DailyPlanView.vue` — `dismissReason` and `dismissNote` refs map to `DismissRequest.reason` / `DismissRequest.note`; valid dismiss reasons hardcoded in select: `completed_elsewhere`, `not_relevant`, `postpone`, `blocked`, `other`

### ⚠ UpdatePlanItem (types/dailyPlan.ts)
Changing this interface shape affects:
- `stores/dailyPlanStore.ts` — `reorderItem` sends `{ userPosition }` via `updateItem`
- `services/dailyPlanService.ts` — `updateItem(itemId, data: UpdatePlanItem)` sends as PUT body

## Cross-Domain Dependencies

- `stores/taskStore.ts` — `useDailyPlan` calls `taskStore.fetchList(true)` in parallel with `fetchPlan`; builds `taskMap` from `taskStore.items` to look up `Task` by `DailyPlanItem.taskId`
- `types/index.ts` (re-exports `Task`) — `PlanItemCard` receives `task?: Task` prop for title display and `task.priority` for color bar via `PriorityColors` enum
- `types/enums.ts` — `PlanItemCard` imports `PriorityColors` keyed by `Task.priority`
- `composables/usePolling.ts` — `useDailyPlan` passes `load` to `usePolling` for auto-refresh
- `components/layout/PageHeader.vue` — `DailyPlanView` uses for title/subtitle/actions header slot
- `components/shared/LoadingSpinner.vue` — shown while `loading && !plan`
- `components/shared/EmptyState.vue` — shown when `!hasPlan && !loading`
- `vue-draggable-plus` (VueDraggable) — `DailyPlanView` wraps each group's items; `@update:model-value` triggers `handleReorder`
- `router/index.ts` — lazy-loads `DailyPlanView` at route `{ path: '/plan', name: 'plan' }`; `openTask` navigates to `{ name: 'task-detail', params: { id: item.taskId } }`

## API Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/v1/daily-plan?date=` | Fetch plan for date (default today) |
| POST | `/api/v1/daily-plan/generate?date=` | Trigger AI generation |
| PUT | `/api/v1/daily-plan/items/:id` | Update position/duration overrides |
| POST | `/api/v1/daily-plan/items/:id/complete` | Mark item complete |
| POST | `/api/v1/daily-plan/items/:id/dismiss` | Dismiss with reason/note |
