# Ingest Backend System

> `ingestbus` is a pure business-layer orchestrator for the email ingestion pipeline. It has no app-layer counterpart and no database of its own. It coordinates five other domain buses (`rawinputbus`, `emailbus`, `taskbus`, `contextbus`, `clarificationbus`) plus an AI extractor (`extractor.Extractor`) to transform a raw RFC 5322 email string into stored records, tasks, context events, and clarification items. The entry point is `smtpbus`, which calls `ingestbus.Business.ProcessEmail()` on every received email. A secondary entry point `Reprocess()` allows re-running the pipeline against an existing `raw_input` record (e.g. after a failure).

---

## Core Types

### `extractor.ContextRef` (extractor/anthropic.go)

```go
type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}
```

Lightweight snapshot of an active context passed to the AI prompt so the extractor can suggest a match.

---

### `extractor.ActionItem` (extractor/anthropic.go)

```go
type ActionItem struct {
    Title           string   `json:"title"`
    Description     string   `json:"description"`
    Priority        string   `json:"priority"`
    Interpretations []string `json:"interpretations,omitempty"`
}
```

One task candidate extracted from an email. When `Interpretations` has more than one entry the pipeline creates an `AmbiguousAction` clarification.

---

### `extractor.Deadline` (extractor/anthropic.go)

```go
type Deadline struct {
    Description string `json:"description"`
    Date        string `json:"date"`
    IsAmbiguous bool   `json:"is_ambiguous,omitempty"`
}
```

A deadline mentioned in the email. When `IsAmbiguous` is true the pipeline creates an `AmbiguousDeadline` clarification.

---

### `extractor.EmailExtraction` (extractor/anthropic.go)

```go
type EmailExtraction struct {
    Summary                  string       `json:"summary"`
    SenderName               string       `json:"sender_name"`
    SenderDomain             string       `json:"sender_domain"`
    ActionItems              []ActionItem `json:"action_items"`
    Deadlines                []Deadline   `json:"deadlines"`
    SuggestedContextKeywords []string     `json:"suggested_context_keywords"`
    Sentiment                string       `json:"sentiment"`
    SuggestedContextID       *string      `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64      `json:"context_confidence,omitempty"`
    SuggestNewContext        bool         `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string       `json:"suggested_context_title,omitempty"`
}
```

Full AI output for one email. Drives all downstream pipeline decisions (context matching, task creation, context event creation, clarification generation).

---

### `extractor.Extractor` interface (extractor/anthropic.go)

```go
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
}
```

Abstraction over the AI call. Production implementation: `AnthropicExtractor`. Test implementation: `MockExtractor`.

---

### `ingestbus.ParsedEmail` (parse.go)

```go
type ParsedEmail struct {
    MessageID   string
    FromAddress string
    FromName    string
    ToAddress   string
    Subject     string
    BodyText    string
    BodyHTML    string
}
```

Intermediate struct produced by RFC 5322 parsing before the email record is stored. Not persisted; only used within `processRawInput`.

---

### `ingestbus.Business` (ingestbus.go)

```go
type Business struct {
    log              *logger.Logger
    rawInputBus      *rawinputbus.Business
    emailBus         *emailbus.Business
    taskBus          *taskbus.Business
    contextBus       *contextbus.Business
    clarificationBus *clarificationbus.Business
    extractor        extractor.Extractor
}
```

The orchestrator. Holds no state beyond its injected dependencies.

---

## File Map

### Core (ingestbus)

- `business/domain/ingestbus/ingestbus.go` — **NewBusiness()**, **ProcessEmail()**, **Reprocess()**, `processRawInput()`, `matchContextByKeywords()` — Pipeline orchestration: wires raw-input → email → AI extraction → context matching → task creation → context event → clarification generation.
- `business/domain/ingestbus/parse.go` — **parseEmail()**, `parseEmailEntity()** — RFC 5322 parsing via `go-message/mail`; produces `ParsedEmail`.

### Extractor sub-package

- `business/domain/ingestbus/extractor/anthropic.go` — **NewAnthropicExtractor()**, **ExtractEmail()** — Defines all shared types (`ContextRef`, `ActionItem`, `Deadline`, `EmailExtraction`, `Extractor` interface) and production AI implementation using `anthropic-sdk-go`.
- `business/domain/ingestbus/extractor/mock.go` — **ExtractEmail()** — `MockExtractor` for testing; returns a pre-configured `EmailExtraction` or error.

### Tests

- `business/domain/ingestbus/ingestbus_test.go` — Integration test stub (requires live DB). Documents four test scenarios: ambiguous deadline clarifications, new-context clarifications, low-confidence context-assignment clarifications, ambiguous-action clarifications.

### Caller (smtpbus)

- `business/domain/smtpbus/smtpbus.go` — **NewServer()**, **ListenAndServe()**, **Data()** — SMTP server (go-smtp). `session.Data()` is the single production entry point that calls `ingestBus.ProcessEmail()`.

### Wire-up

- `api/services/planner/main.go` — Constructs all five dependency buses, calls `extractor.NewAnthropicExtractor()`, then `ingestbus.NewBusiness(...)`. Only instantiated when `cfg.SMTP.Enabled` is true.

---

## Pipeline Step Reference

The 10-step pipeline inside `processRawInput`:

| Step | Action | Bus / Function |
|------|--------|---------------|
| 1 | Store raw_input as `pending` | `rawinputBus.Create()` |
| — | Mark `processing` | `rawinputBus.MarkProcessing()` |
| 2 | Parse RFC 5322 email | `parseEmail()` → `ParsedEmail` |
| 3 | Dedup check by Message-ID | `emailBus.QueryByMessageID()` |
| 4 | Store email record | `emailBus.Create()` |
| 5 | Fetch active contexts (up to 50) | `contextBus.Query()` |
| 6 | AI extraction | `extractor.ExtractEmail()` |
| 7 | Context matching (ID → keyword → auto-create) | `contextBus.QueryByID()`, `contextBus.Create()` |
| 7a | Clarifications: new_context, context_assignment, ambiguous_action, ambiguous_deadline | `clarificationBus.Create()` |
| 8 | Create tasks from action items | `taskBus.Create()` |
| 9 | Create context event | `contextBus.AddEvent()` |
| 10 | Mark raw_input `processed` (or `failed`) | `rawinputBus.MarkProcessed()` / `rawinputBus.MarkFailed()` |

---

## Impact Callouts

### ⚠ `extractor.EmailExtraction` (extractor/anthropic.go)

Changing the shape of this struct affects:

- `extractor/anthropic.go` — JSON unmarshalled from Anthropic API response; field names must match the prompt schema exactly
- `business/domain/ingestbus/ingestbus.go` — every field is read in `processRawInput`: `ActionItems`, `Deadlines`, `SuggestedContextID`, `ContextConfidence`, `SuggestNewContext`, `SuggestedContextTitle`, `SuggestedContextKeywords`, `Sentiment`, `Summary`
- `extractor/mock.go` — `MockExtractor.Result` field; tests must set the relevant fields to exercise pipeline branches
- If the AI prompt schema changes, the `ExtractEmail()` prompt string in `anthropic.go` must be updated in lockstep

### ⚠ `extractor.ActionItem` (extractor/anthropic.go)

- `business/domain/ingestbus/ingestbus.go` — `item.Title`, `item.Description`, `item.Priority`, `item.Interpretations` all read in step 8 (task creation) and step 7a (ambiguous-action clarification)
- `taskbus.NewTask` is populated from `ActionItem`; if task fields are added (e.g. energy override), the mapping here must change

### ⚠ `extractor.Deadline` (extractor/anthropic.go)

- `business/domain/ingestbus/ingestbus.go` — `dl.IsAmbiguous`, `dl.Description`, `dl.Date` read during ambiguous-deadline clarification generation (step 7a)

### ⚠ `extractor.Extractor` interface (extractor/anthropic.go)

Adding or changing the method signature affects:

- `extractor/anthropic.go` — `AnthropicExtractor` must implement the new signature
- `extractor/mock.go` — `MockExtractor` must implement the new signature
- `business/domain/ingestbus/ingestbus.go` — call site `b.extractor.ExtractEmail(...)` must match

### ⚠ `ingestbus.ParsedEmail` (parse.go)

- `business/domain/ingestbus/ingestbus.go` — all fields consumed in `processRawInput`: `MessageID` (dedup), `FromAddress`, `FromName`, `ToAddress`, `Subject`, `BodyText`, `BodyHTML` → mapped onto `emailbus.NewEmail`
- `extractor.ExtractEmail()` call passes `parsed.Subject`, `parsed.BodyText`, `parsed.FromAddress`
- `parseEmailEntity()` also produces `ParsedEmail` (for direct MIME entity parsing path)

### ⚠ `ingestbus.Business` struct / `NewBusiness()` signature (ingestbus.go)

- `api/services/planner/main.go` — constructs `ingestbus.Business` directly; adding a dependency requires updating this call site
- `business/domain/smtpbus/smtpbus.go` — holds `*ingestbus.Business`; only calls `ProcessEmail()`, so structural changes to `Business` do not affect smtpbus unless the method signature changes

### ⚠ Clarification kind usage (ingestbus.go → clarificationkind)

`ingestbus` creates clarification items of kinds: `NewContext`, `ContextAssignment`, `AmbiguousAction`, `AmbiguousDeadline`. If any of these kinds are removed from `clarificationkind`, the pipeline will panic at `clarificationkind.MustParse()` (though `ingestbus` uses the var constants directly, not `MustParse`). If `clarificationbus.NewClarificationItem` fields change, all four creation sites in `ingestbus.go` must be updated.

---

## Routes

`ingestbus` has no HTTP routes. It is invoked exclusively via `smtpbus` (SMTP DATA command) or indirectly through `rawinputapp`'s reprocess endpoint.

The related REST endpoint that surfaces reprocessing is in `rawinputapp`:

| Method | Path | Handler | Notes |
|--------|------|---------|-------|
| POST | `/api/v1/raw-inputs/{raw_input_id}/reprocess` | `rawinputapp.reprocess` | Verifies record exists, then calls `ingestbus.Reprocess()` which marks processing, re-runs the full pipeline, and marks processed/failed |

> `ingestbus.Reprocess()` is fully wired through `rawinputapp`'s reprocess handler.

---

## Cross-Domain Dependencies

| Domain | Usage |
|--------|-------|
| `rawinputbus` | Create, MarkProcessing, MarkProcessed, MarkFailed, QueryByID — tracks pipeline lifecycle |
| `emailbus` | QueryByMessageID (dedup), Create (store email record), Update (attach context) |
| `taskbus` | Create — one task per extracted action item |
| `contextbus` | Query (fetch active contexts), QueryByID (verify suggested context), Create (auto-create new context), AddEvent (record email as context event) |
| `clarificationbus` | Create — generates clarification items for ambiguous/low-confidence decisions |
| `extractor` (sub-package) | AI extraction interface; production impl uses `anthropic-sdk-go` |
| `smtpbus` | Caller; holds `*ingestbus.Business`, calls `ProcessEmail()` |
| `rawinputapp` | Full caller; `reprocess` handler verifies record exists then calls `ingestbus.Reprocess()` for full pipeline re-run |
| `business/types/rawinputsource` | `rawinputsource.Email` constant used when creating raw_input |
| `business/types/clarificationkind` | `NewContext`, `ContextAssignment`, `AmbiguousAction`, `AmbiguousDeadline` constants used when creating clarifications |
| `business/types/taskpriority`, `taskstatus`, `taskenergy` | Enum values used when constructing `taskbus.NewTask` |
| `business/sdk/page` | `page.MustParse("1","50")` for fetching active contexts |
| `foundation/logger` | Structured logging throughout pipeline |
| `foundation/sqldb` | `sqldb.ErrDBNotFound` checked for dedup path and context verification |
| `go-message/mail` | RFC 5322 MIME parsing in `parse.go` |
| `anthropic-sdk-go` | LLM API calls in `extractor/anthropic.go` |
