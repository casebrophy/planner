# Clarification System (Frontend)

> Clarification queue domain: surface AI-generated questions requiring user resolution. Items have a kind (context_assignment, stale_task, ambiguous_deadline, etc.), priority score, and status lifecycle (pending → resolved/snoozed/dismissed). The view presents a single-card-at-a-time flow with resolve/snooze/dismiss actions. A pending count badge on the sidebar is polled separately.

## Core Types

```ts
// types/clarification.ts
interface ClarificationItem {
  id: string
  kind: ClarificationKind           // see enums below
  status: ClarificationStatus       // 'pending' | 'snoozed' | 'resolved' | 'dismissed'
  subjectType: string               // e.g. 'task', 'context'
  subjectId: string
  question: string
  claudeGuess?: Record<string, unknown>   // AI's suggested answer
  reasoning?: string
  answerOptions: Record<string, unknown>   // possible answers
  answer?: Record<string, unknown>         // resolved answer
  priorityScore: number
  snoozedUntil?: string
  createdAt: string
  resolvedAt?: string
}

interface ClarificationCountResponse {
  count: number
}
```

```ts
// types/enums.ts (clarification-related)
const ClarificationKind = {
  ContextAssignment: 'context_assignment',
  StaleTask: 'stale_task',
  AmbiguousDeadline: 'ambiguous_deadline',
  NewContext: 'new_context',
  OverlappingContexts: 'overlapping_contexts',
  AmbiguousAction: 'ambiguous_action',
  VoiceReference: 'voice_reference',
  InactivityPrompt: 'inactivity_prompt',
  ContextDebrief: 'context_debrief',
} as const
type ClarificationKind = (typeof ClarificationKind)[keyof typeof ClarificationKind]

const ClarificationStatus = {
  Pending: 'pending', Snoozed: 'snoozed', Resolved: 'resolved', Dismissed: 'dismissed',
} as const
type ClarificationStatus = (typeof ClarificationStatus)[keyof typeof ClarificationStatus]

const ClarificationKindLabels: Record<ClarificationKind, string>  // display labels
const ClarificationKindColors: Record<ClarificationKind, string>  // hex colors per kind
```

## File Map

### Stores
- `stores/clarificationStore.ts` — **useClarificationStore** — custom store (not createCRUDStore); holds `items` (ClarificationItem[]), `currentIndex`, `pendingCount`; computes `currentItem`, `hasNext`, `isEmpty`, `progress`; exposes `fetchQueue`, `fetchPendingCount`, `resolve`, `snooze`, `dismiss`, `goTo`; `removeAndAdvance` keeps index valid after actions

### Services
- `services/clarificationService.ts` — **clarificationService** — hand-written service (not createCRUDService); calls `/api/v1/clarifications` (query), `/api/v1/clarifications/count`, `/api/v1/clarifications/:id/resolve` (POST), `/api/v1/clarifications/:id/snooze` (POST), `/api/v1/clarifications/:id/dismiss` (POST)

### Composables
- `composables/useClarification.ts` — **useClarification** — thin composable; fetches queue on mount, exposes store refs + store actions (resolve/snooze/dismiss/refresh); pending count polling is handled globally in AppSidebar

### Components
- `components/clarifications/ClarificationCard.vue` — **ClarificationCard** — renders a single ClarificationItem: kind label, question, answerOptions, claudeGuess; emits resolve/snooze/dismiss
- `components/clarifications/ClarificationSession.vue` — **ClarificationSession** — wraps the card flow: progress indicator, navigation, empty state

### Views
- `views/ClarificationView.vue` — **ClarificationView** — uses useClarification; renders ClarificationSession with progress and card

## Impact Callouts

### ⚠ ClarificationItem (types/clarification.ts)
Changing this interface shape affects:
- `stores/clarificationStore.ts` — stores ClarificationItem[] in `items`; `resolve` passes `answer: Record<string,unknown>`; `removeAndAdvance` finds by `.id`
- `composables/useClarification.ts` — exposes `currentItem` ref typed as ClarificationItem | null
- `services/clarificationService.ts` — deserializes ClarificationItem from all endpoints; `resolve` sends `{ answer }` body; `snooze` sends `{ hours }` body
- `components/clarifications/ClarificationCard.vue` — binds .kind, .question, .answerOptions, .claudeGuess, .reasoning, .priorityScore
- `components/clarifications/ClarificationSession.vue` — reads .id for key tracking

### ⚠ ClarificationKind (types/enums.ts)
Adding or removing kind values affects:
- `components/clarifications/ClarificationCard.vue` — uses ClarificationKindLabels and ClarificationKindColors for display
- `services/clarificationService.ts` — kind is returned from API; new kinds appear automatically if labels/colors map is updated
- `stores/clarificationStore.ts` — no direct kind check (filter is status-based), but kind is stored in ClarificationItem

### ⚠ ClarificationStatus (types/enums.ts)
Changing status values affects:
- `stores/clarificationStore.ts` — fetchQueue filters by `status: 'pending'` string literal (not enum reference)
- `services/clarificationService.ts` — queryQueue accepts status as string param

## Cross-Domain Dependencies

- `stores/toastStore.ts` — clarificationStore calls toast.success/error on resolve/snooze/dismiss
- `components/layout/AppSidebar.vue` — polls `clarificationStore.fetchPendingCount()` every 60s; reads `pendingCount` for badge display
