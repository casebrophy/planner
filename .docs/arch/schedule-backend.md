# Schedule Backend System

> Unified schedule query. Merges events and time blocks into a single time-ordered list for a date range. App-layer only — no business or store layer (reads from eventbus and timeblockbus directly).

## Core Types

```go
// app/domain/scheduleapp/scheduleapp.go
type ScheduleItem struct {
    Type      string  `json:"type"`                // "event" or "time_block"
    ID        string  `json:"id"`
    Title     string  `json:"title"`
    StartsAt  string  `json:"startsAt"`
    EndsAt    string  `json:"endsAt"`
    AllDay    bool    `json:"allDay,omitempty"`
    Location  *string `json:"location,omitempty"`
    TaskID    *string `json:"taskId,omitempty"`     // time_block only
    Confirmed *bool   `json:"confirmed,omitempty"`  // time_block only
}

type scheduleResponse struct {
    Items []ScheduleItem `json:"items"`
}
```

## File Map

### Handlers
- `app/domain/scheduleapp/scheduleapp.go` — **querySchedule()** GET — parses `?start=&end=` RFC3339 params, queries eventbus + timeblockbus with DateFrom/DateTo filters, merges into `[]ScheduleItem`, sorts by StartsAt
- `app/domain/scheduleapp/route.go` — Routes.Add() wiring, instantiates eventbus + timeblockbus from mux.Config

## Impact Callouts

### ⚠ ScheduleItem (app/domain/scheduleapp/scheduleapp.go)
Changing this struct affects:
- Frontend calendar view — consumes the JSON response
- No other backend files depend on this type

### ⚠ eventbus.Event or timeblockbus.TimeBlock field changes
If fields read by scheduleapp change:
- `scheduleapp.go:84-94` — reads event.ID, Title, StartsAt, EndsAt, AllDay, Location
- `scheduleapp.go:97-109` — reads block.ID, TaskID, StartsAt, EndsAt, Confirmed
- Update the field mapping in the merge loop

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/schedule?start=&end= | querySchedule | API key |

## Cross-Domain Dependencies

- **eventbus** — queries events by date range (DateFrom/DateTo filter)
- **timeblockbus** — queries time blocks by date range (DateFrom/DateTo filter)
- Uses `page.New(1, 200)` for both queries — fetches up to 200 items per type
