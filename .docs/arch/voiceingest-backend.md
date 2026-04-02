# Voice Ingest Backend System

Accepts raw text from voice input (e.g., Siri Shortcut), feeds through the ingestbus pipeline to create tasks and events. Single POST endpoint that validates input, processes through the ingest pipeline, and returns created resource IDs.

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
    TaskIDs  []string `json:"taskIds"`
    EventIDs []string `json:"eventIds"`
}
```

### IngestResult (Business Layer)
```go
type IngestResult struct {
    TaskIDs  []uuid.UUID
    EventIDs []uuid.UUID
}
```
(From ingestbus package — returned by ProcessText)

## File Map

### App (Handlers)
- `app/domain/voiceingestapp/voiceingestapp.go` — **ingest()** — POST /api/v1/ingest/voice, accepts text, validates non-empty, calls ingestbus.ProcessText, converts UUIDs to strings, returns ingestResponse
- `app/domain/voiceingestapp/model.go` — request/response DTOs (ingestRequest, ingestResponse)
- `app/domain/voiceingestapp/route.go` — route registration, wires full ingestbus dependency chain including rawinputbus, emailbus, taskbus, contextbus, clarificationbus, eventbus, and extractor

## Impact Callouts

### ⚠ ingestbus.ProcessText signature
If ProcessText changes return type or parameters, update voiceingestapp.ingest() handler accordingly.

### ⚠ ingestbus.IngestResult fields
If new fields added beyond TaskIDs and EventIDs, update ingestResponse struct and handler response mapping.

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
