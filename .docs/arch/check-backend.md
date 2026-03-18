# Check Backend System

> Health check endpoints for readiness and liveness probes. No business logic — just verifies database connectivity and service availability.

## Core Types

```go
// app/domain/checkapp/checkapp.go

type status struct {
    Status string `json:"status"`
}
```

## File Map

### App (Handlers)
- `app/domain/checkapp/checkapp.go` — **readiness()** — checks DB connection via `sqldb.StatusCheck()`. **liveness()** — returns `{"status":"ok"}` unconditionally.
- `app/domain/checkapp/route.go` — Route registration, no middleware applied.

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| GET | /api/v1/readiness | readiness | None (HandleNoMiddleware) |
| GET | /api/v1/liveness | liveness | None (HandleNoMiddleware) |

## Cross-Domain Dependencies

- **sqldb.StatusCheck** — database connectivity probe
- **web.Encoder** — status implements this via `Encode()`
