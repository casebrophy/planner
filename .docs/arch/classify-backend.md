# Classify Backend System

> The classify domain provides a synchronous HTTP endpoint that assigns open tasks (those lacking a context) to contexts using AI extraction. For each unlinked task it calls the text extractor; high-confidence matches (≥ 0.7) are auto-applied; low-confidence ones produce a `context_assignment` clarification card for user review. There is no business layer or dedicated store — the handler composes `taskbus`, `contextbus`, `clarificationbus`, and `extractor` directly.

## Core Types

### App Handler Struct (`app/domain/classifyapp/classifyapp.go`)
```go
type app struct {
    taskBus          *taskbus.Business
    contextBus       *contextbus.Business
    clarificationBus *clarificationbus.Business
    extractor        extractor.Extractor
}
```

### Response DTO (`app/domain/classifyapp/model.go`)
```go
type ClassifyResult struct {
    Classified            int `json:"classified"`
    ClarificationsCreated int `json:"clarificationsCreated"`
}
```

### Extractor Interface (`business/domain/ingestbus/extractor/model.go`)
```go
type Extractor interface {
    ExtractEmail(ctx context.Context, subject, bodyText, fromAddress string, activeContexts []ContextRef) (EmailExtraction, error)
    ExtractText(ctx context.Context, text string, activeContexts []ContextRef) (TextExtraction, error)
}

type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type TextExtraction struct {
    Summary                  string           `json:"summary"`
    ActionItems              []ActionItem     `json:"action_items"`
    Deadlines                []Deadline       `json:"deadlines"`
    Events                   []ExtractedEvent `json:"events"`
    Notes                    []ExtractedNote  `json:"notes"`
    SuggestedContextKeywords []string         `json:"suggested_context_keywords"`
    SuggestedContextID       *string          `json:"suggested_context_id,omitempty"`
    ContextConfidence        float64          `json:"context_confidence,omitempty"`
    SuggestNewContext        bool             `json:"suggest_new_context,omitempty"`
    SuggestedContextTitle    string           `json:"suggested_context_title,omitempty"`
}
```

### Clarification Options (`business/domain/clarificationbus/options.go`)
```go
type ContextRef struct {
    ID    string `json:"id"`
    Title string `json:"title"`
}

type ContextAssignmentOptions struct {
    SuggestedContext  string       `json:"suggested_context"`
    Confidence        float64      `json:"confidence"`
    AvailableContexts []ContextRef `json:"available_contexts"`
}
```

## File Map

### App Layer
- `app/domain/classifyapp/model.go` — **ClassifyResult** — response DTO with `Encode()` for `web.Encoder`
- `app/domain/classifyapp/classifyapp.go` — **classify()** — POST /api/v1/tasks/classify handler; queries unlinked tasks, fetches contexts, calls extractor per task, auto-assigns or creates clarification
- `app/domain/classifyapp/route.go` — **Routes.Add()** — wires taskbus, contextbus, clarificationbus, extractor (Claude + optional Ollama failover); registers route with auth middleware

### No Dedicated Business or Store Layer
The classify domain has no `business/domain/classifybus/` or store. It is a pure composition of existing buses and the extractor. The handler lives entirely in `app/domain/classifyapp/`.

## Impact Callouts

### ⚠ ClassifyResult (`app/domain/classifyapp/model.go`)
Changing this struct shape affects:
- `app/domain/classifyapp/classifyapp.go` — returned directly from `classify()`
- Any frontend consumer of `POST /api/v1/tasks/classify` — JSON field names change

### ⚠ extractor.Extractor interface (`business/domain/ingestbus/extractor/model.go`)
Adding/changing a method affects:
- `app/domain/classifyapp/classifyapp.go` — calls `ExtractText()` per task
- `app/domain/classifyapp/route.go` — wires the concrete extractor (claudecli / ollama / failover)
- `app/domain/mcpapp/mcpapp.go` — also calls `ExtractText()` for background classify tool
- `business/domain/ingestbus/ingestbus.go` — calls both `ExtractText()` and `ExtractEmail()`
- All extractor implementations (`extractor/claudecli.go`, `extractor/ollama.go`, `extractor/mock.go`) must implement the new method

### ⚠ extractor.TextExtraction (`business/domain/ingestbus/extractor/model.go`)
Changing this struct affects:
- `app/domain/classifyapp/classifyapp.go` — reads `SuggestedContextID`, `ContextConfidence`
- `app/domain/mcpapp/mcpapp.go` — reads same fields in background goroutine
- `business/domain/ingestbus/ingestbus.go` — reads all fields in `processTextInput()`
- Extractor implementations that produce this struct

### ⚠ clarificationbus.ContextAssignmentOptions (`business/domain/clarificationbus/options.go`)
Changing this struct affects:
- `app/domain/classifyapp/classifyapp.go` — marshals to `AnswerOptions` on low-confidence tasks
- `app/domain/mcpapp/mcpapp.go` — same, inside background goroutine
- `business/domain/ingestbus/ingestbus.go` — email and text paths both marshal this struct
- Frontend `ClarificationCard` — deserializes `answer_options` JSON for `context_assignment` kind; field renames break display

### ⚠ extractor.ContextRef (`business/domain/ingestbus/extractor/model.go`)
This struct is used to pass context candidates to the extractor. It is structurally identical to `clarificationbus.ContextRef` but is a separate type to avoid import cycles.
- `app/domain/classifyapp/classifyapp.go` — builds `[]extractor.ContextRef` from contextbus results, converts to `[]clarificationbus.ContextRef` for options JSON
- `app/domain/mcpapp/mcpapp.go` — same pattern
- `business/domain/ingestbus/ingestbus.go` — same pattern in both `processEmail()` and `processTextInput()`

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | /api/v1/tasks/classify | `classify()` | API key (`X-API-Key`) |

## Cross-Domain Dependencies

- **taskbus** — queries open tasks (`QueryFilter{Status: &Open}`), updates task `ContextID` on high-confidence match
- **contextbus** — queries active contexts to build `ctxRefs`, calls `QueryByID` to verify suggested context exists
- **clarificationbus** — creates `NewClarificationItem` with `Kind: ContextAssignment` on low-confidence matches; uses `clarificationbus.ContextAssignmentOptions` for typed `AnswerOptions` JSON
- **ingestbus/extractor** — `Extractor` interface (claudecli + optional ollama failover) for AI text classification
- **clarificationkind** type enum — `clarificationkind.ContextAssignment` constant
- **taskstatus** type enum — `taskstatus.Open` filter constant
- **contextstatus** — `contextbus.Active` filter constant
- **mux.Config** — provides `DB`, `Log`, `ClaudeCLI`, `APIKey`, `OllamaEnabled`, `OllamaURL`, `OllamaModel`
