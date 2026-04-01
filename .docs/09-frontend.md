# Frontend design

Single Vue 3 codebase with two shells (web sidebar, mobile tab bar) sharing all components, stores, and API calls; Capacitor packages it as a native iOS app.

---

## Navigation structure

### Web sidebar
```
Dashboard / Today / Tasks / Contexts / Transactions / Clarifications / Capture / Search / Settings
```

### Mobile tab bar (5 tabs)
```
[Capture]  [Today]  [Contexts]  [Search]  [Settings]
```

---

## Shared components

### Data display

| Component | Purpose |
|-----------|---------|
| `TaskCard` | Title, priority, status, due date, context tag; opens TaskDetail |
| `ContextCard` | Title, summary excerpt, open task count, last activity |
| `NoteItem` | Timestamped note with source indicator (user, claude, system, email, voice) |
| `TransactionRow` | Merchant, amount, date, category, context tag; swipeable on mobile |
| `PlanItemCard` | Task within daily plan: title, AI duration, group badge, drag handle, dismiss/complete actions |
| `PlanGroupHeader` | Group separator in daily plan (e.g. "Errands", "Deep Work", context title) |
| `EventCard` | Fixed commitment: title, time range, location; not checkable |
| `TimeBlock` | Scheduled task slot with time range; used in calendar view (Phase 7b) |
| `EventTimelineItem` | Single context event log entry; polymorphic by `kind` |

### Input

| Component | Purpose |
|-----------|---------|
| `CaptureInput` | Text + image capture; native camera on mobile, file picker on web |
| `TaskForm` | Full task create/edit (title, description, priority, due date, energy, duration, context, tags) |
| `ContextForm` | Create/edit context (title, description, tags, status) |
| `EventForm` | Create/edit event (title, description, start/end, location, all-day, context) |
| `SearchBar` | Text input wired to `search_semantic`; recent searches, filter chips |

### Feedback

| Component | Purpose |
|-----------|---------|
| `ProcessingStatus` | Tracks capture pipeline stages (pending → processing → processed) |
| `ConfirmationPrompt` | Claude reference-resolution confirm UI; bottom sheet on mobile, inline card on web |
| `ReviewFlag` | Badge/banner for items needing attention (unmatched transactions, low-confidence assignments) |

---

## Routes

| Path | View | Shell |
|------|------|-------|
| `/` | Redirect → `/dashboard` (web) or `/capture` (mobile) | both |
| `/dashboard` | Dashboard | web |
| `/capture` | CaptureScreen | mobile primary, web secondary |
| `/today` | TodayView | mobile primary, web accessible |
| `/tasks` | TaskBoard | both |
| `/tasks/:id` | TaskDetail | both |
| `/contexts` | ContextBoard | both |
| `/contexts/:id` | ContextDetail | both |
| `/transactions` | TransactionBoard | both |
| `/clarifications` | ClarificationView | both |
| `/schedule` | ScheduleView | web primary (future — Phase 7) |
| `/search` | SearchView | both |
| `/settings` | Settings | both |

Shell detection: `useShell()` composable returns `isMobile` ref. Currently uses `window.innerWidth < 768` (matches Tailwind `md:`). When Capacitor is added, swap to `Capacitor.isNativePlatform() || window.innerWidth < 768`. Router `/` redirect uses `window.innerWidth` directly (runs before component mount).

---

## PWA

- `vite-plugin-pwa` generates service worker (workbox, `autoUpdate` strategy)
- `public/manifest.json` — standalone display, dark theme colors
- `public/icons/` — SVG icons (192, 512)
- iOS meta tags: `apple-mobile-web-app-capable`, `black-translucent` status bar
- Go static server sets `Cache-Control: no-cache` on `sw.js` and `manifest.json`
- Capacitor path: add `@capacitor/core`, init, sync — PWA manifest/SW are ignored in native builds

---

## Pinia stores

| Store | Responsibility |
|-------|---------------|
| `useTaskStore` | Tasks CRUD, filter state, optimistic updates |
| `useContextStore` | Contexts CRUD, summary, event timeline |
| `useTransactionStore` | Transactions, review queue |
| `useScheduleStore` | Time blocks, calendar availability |
| `useCaptureStore` | In-progress capture, recent submissions, processing status |
| `useSessionStore` | Recent activity window, session ID injected into every API request |
| `useSearchStore` | Query state, results, search history |
| `useSettingsStore` | API key, preferences, notification config |
| `useClarificationStore` | Clarification queue, pending count, resolve/snooze |
| `useTagStore` | Tags CRUD |
| `useToastStore` | Toast notification queue |

---

## api.ts structure

```typescript
const client = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  headers: { 'X-API-Key': import.meta.env.VITE_API_KEY }
})

client.interceptors.request.use(config => {
  config.headers['X-Session-ID'] = useSessionStore().currentSessionId
  return config
})

export const api = {
  tasks:        { list, get, create, update, complete, delete: remove },
  contexts:     { list, get, create, update },
  transactions: { list, review, assignContext },
  capture:      { submit },           // multipart: image + text
  search:       { semantic },
  schedule:     { list, createBlock, confirmBlock },
  activity:     { recent },
  notes:        { list, add }
}
```

---

## Build commands

```bash
# Web dev
vite dev

# Web production
vite build

# iOS (Capacitor)
vite build && npx cap sync ios && npx cap open ios
# or: npx cap run ios
```

Environment vars: `VITE_API_BASE_URL`, `VITE_API_KEY` (in `.env` for web, `.env.mobile` for Capacitor).
