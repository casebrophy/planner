# Voice Ingest Backend System

Accepts raw text from voice input (e.g., Siri Shortcut), enqueues asynchronously for processing through the ingestbus pipeline to create tasks and events. Single POST endpoint that validates input, stores raw input record, and immediately returns the raw_input_id (does not wait for pipeline completion).

## Core Types

### ingestRequest (App Layer)
```go
type ingestRequest struct {
    Text string `json:"text"`
}
```

### ingestResponse (App Layer)
```go
type ingestResponse struct {
    RawInputID string `json:"rawInputId"`
}
```

## File Map

### App (Handlers)
- `app/domain/voiceingestapp/voiceingestapp.go` — **ingest()** — POST /api/v1/ingest/voice, accepts text, validates non-empty, calls ingestbus.EnqueueText (returns immediately with raw_input_id), returns ingestResponse
- `app/domain/voiceingestapp/model.go` — request/response DTOs (ingestRequest, ingestResponse with RawInputID)
- `app/domain/voiceingestapp/route.go` — route registration, wires ingestbus dependency chain including rawinputbus, emailbus, taskbus, contextbus, clarificationbus, eventbus, and extractor

## Impact Callouts

### ⚠ ingestbus.EnqueueText signature
If EnqueueText changes return type (uuid.UUID) or parameters, update voiceingestapp.ingest() handler accordingly.

### ⚠ Async processing — no response blocking
The handler does NOT wait for pipeline completion. It enqueues the work and returns the raw_input_id immediately. The client must poll the raw_input endpoint to check pipeline status.

### ⚠ Dependency chain in route.go
All domain stores (rawinput, email, task, context, clarification, event) and extractor must be instantiated and passed to ingestbus.NewBusiness — if ingestbus constructor signature changes, all wiring must be updated.

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | /api/v1/ingest/voice | ingest | API key |

## Cross-Domain Dependencies

**Wired in route.go:**
- **rawinputbus** — stores raw input records
- **emailbus** — email domain (part of ingest pipeline)
- **taskbus** — creates tasks from ingested text
- **contextbus** — context management for ingest
- **clarificationbus** — clarification requests during ingest
- **eventbus** — creates events from ingested text
- **extractor** (ingestbus.extractor) — extracts structured data from text via Claude API

All dependencies instantiated with stores (DB layer) and wired into ingestbus.Business on route registration.
