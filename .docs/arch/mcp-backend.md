# MCP Backend System

The MCP (Model Context Protocol) domain is a unified HTTP interface that exposes the entire planner application to Claude via standardized JSON-RPC 2.0 messages. It acts as a gateway, marshaling requests from Claude agents into business layer operations across all domains (tasks, contexts, clarifications, events, etc.), and serializing responses back to JSON.

This is the **primary integration point** for MCP-connected Claude agents to interact with the planner — every tool call from Claude flows through this handler and dispatches to the appropriate business layer.

## Core Types

### JSON-RPC 2.0 Protocol Types

```go
// HTTP POST body: JSON-RPC 2.0 request
type rpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`        // "2.0"
	ID      any             `json:"id"`             // Request ID (string, number, or null)
	Method  string          `json:"method"`         // RPC method name ("tools/call", "tools/list", "initialize")
	Params  json.RawMessage `json:"params,omitempty"`  // Method-specific params
}

// HTTP response: JSON-RPC 2.0 response
type rpcResponse struct {
	JSONRPC string    `json:"jsonrpc"`        // "2.0"
	ID      any       `json:"id"`             // Echo of request ID
	Result  any       `json:"result,omitempty"`  // Success result (any type, determined by method)
	Error   *rpcError `json:"error,omitempty"`   // Error object if error occurred
}

// Error envelope
type rpcError struct {
	Code    int    `json:"code"`      // -32700, -32602, -32601, or custom
	Message string `json:"message"`   // Human-readable error message
}
```

### MCP Tool Types

```go
// Tool definition (broadcast to client in tools/list response)
type toolDef struct {
	Name        string `json:"name"`         // Unique tool name ("create_task", "list_events")
	Description string `json:"description"` // Human-readable purpose
	InputSchema any    `json:"inputSchema"` // JSON Schema for input validation
}

// Tool call params (received in tools/call request)
type toolCallParams struct {
	Name      string          `json:"name"`           // Which tool to call
	Arguments json.RawMessage `json:"arguments,omitempty"` // Tool-specific args (JSON)
}

// Tool execution result
type toolResult struct {
	Content []toolContent `json:"content"` // Array of content blocks
	IsError bool          `json:"isError,omitempty"` // true if tool execution failed
}

type toolContent struct {
	Type string `json:"type"` // "text"
	Text string `json:"text"` // Result text
}
```

### Server Initialization

```go
// Response to initialize request
type initializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"` // "2025-03-26"
	ServerInfo      serverInfo `json:"serverInfo"`
	Capabilities    any        `json:"capabilities"`    // {"tools": {}}
}

type serverInfo struct {
	Name    string `json:"name"`    // "planner"
	Version string `json:"version"` // "0.1.0"
}
```

### Handler Struct

```go
type app struct {
	taskBus          *taskbus.Business
	contextBus       *contextbus.Business
	emailBus         *emailbus.Business
	eventBus         *eventbus.Business
	timeBlockBus     *timeblockbus.Business
	clarificationBus *clarificationbus.Business
	threadBus        *threadbus.Business
	observationBus   *observationbus.Business
	debriefBus       *debriefbus.Business
	dailyPlanBus     *dailyplanbus.Business
	noteBus          *notebus.Business
	tagBus           *tagbus.Business
	activityLogBus   *activitylogbus.Business
	extractor        extractor.Extractor  // Claude Code extractor for context classification
}
```

### Typed Clarification Option Structs (clarificationbus/options.go)

These typed structs replace raw `map[string]any` at all clarification creation sites:

```go
// ContextRef is a lightweight context pointer used in clarification options.
// Defined in clarificationbus to avoid dependency on ingestbus/extractor.
type ContextRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// ContextAssignmentOptions is the typed AnswerOptions for context_assignment clarifications.
type ContextAssignmentOptions struct {
	SuggestedContext  string       `json:"suggested_context"`
	Confidence        float64      `json:"confidence"`
	AvailableContexts []ContextRef `json:"available_contexts"`
}

// NewContextOptions is the typed AnswerOptions for new_context clarifications.
type NewContextOptions struct {
	ContextID string `json:"context_id"`
	Title     string `json:"title"`
}

// AmbiguousActionOptions is the typed AnswerOptions for ambiguous_action clarifications.
type AmbiguousActionOptions struct {
	Interpretations []string `json:"interpretations"`
}

// AmbiguousDeadlineOptions is the typed AnswerOptions for ambiguous_deadline clarifications.
type AmbiguousDeadlineOptions struct {
	Description string `json:"description"`
	RawDate     string `json:"raw_date"`
}
```

## File Map

### Route Registration
- `route.go` — **Routes.Add()** — Wires MCP handler to HTTP router at `POST /mcp`, instantiates all business layer buses, applies auth middleware (API key)

### Handler
- `mcpapp.go` — HTTP handler + JSON-RPC dispatcher — implements all tool handlers, dispatches on method name

### Tool Definitions
- `tools.go` — Static array of `toolDef` tool definitions (30+ tools), each with name, description, and JSON schema

### Models
- `model.go` — JSON-RPC protocol types (rpcRequest, rpcResponse, rpcError), MCP types (toolDef, toolCallParams, toolResult), initialization response types

## Handler Methods (Tool Implementations)

The `app.handle()` dispatcher decodes the rpcRequest, dispatches on req.Method, and routes to the appropriate handler:

### Initialize & Metadata
- **handle()** — Main HTTP handler; routes on req.Method to dispatch, wraps responses in rpcResponse envelope
- **toolListTools()** — Returns static `tools` array
- **initialize()** — Inline in handle(); returns initializeResult

### Task Tools
- **toolCreateTask()** — Parses create_task args, calls taskBus.Create(), returns created task ID
- **toolListTasks()** — Parses list_tasks filters (status, priority, context_id, page), calls taskBus.Query(), returns paginated task list
- **toolGetTask()** — Parses task_id, calls taskBus.QueryByID(), returns full task with threads
- **toolUpdateTask()** — Parses update_task args, calls taskBus.Update(), returns updated task
- **toolCompleteTask()** — Marks task as done, calls taskBus.Update()

### Context Tools
- **toolCreateContext()** — Parses create_context args, calls contextBus.Create()
- **toolGetContext()** — Parses context_id, calls contextBus.QueryByID()
- **toolListContexts()** — Lists all active/paused/closed contexts via contextBus.Query()
- **toolUpdateContext()** — Updates context title/description/status/summary

### Email Tools
- **toolListEmails()** — Lists ingested emails with optional filters (from_address, context_id)
- **toolGetEmail()** — Fetches full email details by ID

### Clarification Tools (AI-generated tasks awaiting user confirmation)
- **toolGetClarificationQueue()** — Returns pending/snoozed clarifications, optionally filtered by kind (context_assignment, stale_task, etc.)
- **toolResolveClarification()** — User provides answer to clarification, triggers resolution side-effects (updates task context, marks task as blocked, etc.)
- **toolSnoozeClarification()** — Temporarily hides clarification for N hours

### Thread Tools (task/context audit trail)
- **toolAddThreadEntry()** — Appends update/blocker/decision/milestone entry to task or context thread
- **toolGetThread()** — Returns paginated thread history for a task or context

### Observation Tools (retrospective/lessons-learned tracking)
- **toolRecordOutcome()** — Records observation (duration_accuracy, blocker_profile, lesson, etc.) for a task or context
- **toolGetOutcomeObservations()** — Queries observations for a task or context

### Event & Calendar Tools
- **toolCreateEvent()** — Creates calendar event (meeting, deadline, milestone)
- **toolListEvents()** — Lists events in optional date range, optionally filtered by context
- **toolGetEvent()** — Fetches event details by ID
- **toolUpdateEvent()** — Updates event fields
- **toolDeleteEvent()** — Soft-deletes event

### Daily Plan & Schedule Tools
- **toolGetDailyPlan()** — Fetches AI-generated daily plan for a date (groups tasks, provides prioritization)
- **toolGenerateDailyPlan()** — Triggers plan generation
- **toolGetSchedule()** — Returns merged calendar of events + time blocks for a date range
- **toolCreateTimeBlock()** — Schedules a task into a specific time slot
- **toolConfirmTimeBlock()** — User confirms/locks in a proposed time block

### Task Dependency Tools
- **toolAddTaskDependency()** — Makes one task dependent on another (task A blocks task B)
- **toolRemoveTaskDependency()** — Removes dependency
- **toolGetTaskDependencies()** — Returns upstream (blocks this task) and downstream (this task blocks)

### Note Tools
- **toolCreateNote()** — Creates a note with optional tags and context
- **toolSearchNotes()** — Full-text search notes by keyword, tag, or context
- **toolListNotesByTag()** — Lists all notes with a specific tag

### Activity & Streak Tools
- **toolLogActivity()** — Records activity entry for habit tracking (e.g., "5km run", "30min study")
- **toolGetStreaks()** — Returns streak and frequency info for a task or note

### Context Classification (Auto-linking Tasks)
- **toolClassifyTasks()** — Queries unlinked tasks and active contexts, returns immediately with unlinked count, then processes in a background goroutine (using `context.Background()` to avoid request cancellation). For each task: runs AI extraction, auto-links high-confidence (>=70%) matches via taskBus.Update(), creates `ContextAssignmentOptions`-typed clarifications for low-confidence matches. busCtxRefs conversion (`[]clarificationbus.ContextRef`) is hoisted before the goroutine launch to avoid repeated work per task.

### Inference Context
- **toolGetInferenceContext()** — Returns pre-assembled context for inference pipelines (daily_plan, email_extraction, text_extraction, thread_classification)

## Impact Callouts

### ⚠ rpcRequest / rpcResponse (model.go)
Changing JSON-RPC envelope structure affects:
- `mcpapp.go:handle()` — Must unmarshal rpcRequest and marshal rpcResponse
- Route registration — Auth middleware and MCP handler depend on envelope structure
- Client integration — Claude agents expect exact JSON-RPC 2.0 protocol; breaking changes cause MCP transport failure

### ⚠ toolCallParams (model.go)
Changing tool params structure affects:
- `mcpapp.go:handle()` — Must unmarshal toolCallParams from rpcRequest.Params
- All tool handlers — Must parse toolCallParams.Arguments

### ⚠ toolDef / tools array (tools.go)
Changing tool definitions affects:
- `mcpapp.go:handle()` — Switch case routes on toolCallParams.Name; adding tool requires new case + handler
- Claude agent integration — tools array is broadcast in tools/list response; agents depend on exact schema

### ⚠ app struct (mcpapp.go)
Adding/removing business layer bus field affects:
- `route.go:Routes.Add()` — Must instantiate and wire new bus
- All tool handlers — May need to call new bus for new functionality
- Example: Adding `suggestionBus` for AI suggestions would require updating both locations

### ⚠ Tool Handler Pattern (mcpapp.go, all toolX methods)
Each tool handler:
1. Unmarshals JSON args into Go struct
2. Calls appropriate business bus method(s)
3. Returns toolResult with Content text

Changing error handling or response format affects:
- Client integration — Agents parse Content[0].Text to extract results
- Clarification creation — Some handlers create clarificationBus items; if error handling changes, may not surface user confusion

### ⚠ ContextAssignmentOptions / ContextRef (clarificationbus/options.go)
These typed structs replace raw `map[string]any` for clarification AnswerOptions. Changing their shape affects:
- `mcpapp.go:toolClassifyTasks()` — Marshals `ContextAssignmentOptions` into `NewClarificationItem.AnswerOptions`
- `app/domain/classifyapp/` — Also creates context_assignment clarifications; must use same struct
- `app/domain/dailyplanapp/` — Creates clarifications during plan generation; must use same struct
- Frontend `ClassifyDialog.vue` — Reads `AnswerOptions` fields by JSON key; field renames break the UI
- `classifyService.ts` / `dailyPlanService.ts` — TypeScript codegen types (`tygo`) mirror these structs; regenerate after field changes

### ⚠ busCtxRefs Conversion (mcpapp.go:toolClassifyTasks)
Context refs are converted from `[]extractor.ContextRef` → `[]clarificationbus.ContextRef` **before** the background goroutine is launched to avoid N redundant conversions.

- `mcpapp.go:toolClassifyTasks()` — Hoisted conversion used inside goroutine for all low-confidence tasks
- If `clarificationbus.ContextRef` fields change, both the conversion loop and all callers must update

### ⚠ Auth Middleware (route.go)
- All MCP routes require `X-API-Key` header matching PLANNER_AUTH_API_KEY
- Changing auth scheme requires updating both `route.go` middleware setup and sidecar systemd unit configuration

## Routes

| Method | Path | Handler | Requires Auth |
|--------|------|---------|--------|
| POST | /mcp | app.handle() | Yes (X-API-Key) |

The handler dispatches on JSON-RPC method in request body:
- `initialize` → returns initializeResult
- `notifications/initialized` → acknowledges client ready
- `tools/list` → returns tools array
- `tools/call` → parses toolCallParams, dispatches to tool handler

## Cross-Domain Dependencies

The MCP handler orchestrates operations across **all business domains**:

- **taskbus** — Create/update/query tasks; task dependency management; activity logging
- **contextbus** — Create/query/update contexts (project/area tracking)
- **clarificationbus** — Retrieve clarification queue; resolve user answers; create auto-generated clarifications for context classification
- **eventbus** — Create/query/update calendar events and deadlines
- **timeblockbus** — Schedule tasks into time slots
- **dailyplanbus** — Generate and fetch AI-optimized daily plans (groups tasks by priority/context)
- **threadbus** — Append and retrieve task/context audit trails
- **observationbus** — Record and query retrospective lessons learned, duration accuracy, blocker profiles
- **emailbus** — Query ingested emails
- **notebus** — Create, search, and tag notes
- **tagbus** — Manage note tags
- **activitylogbus** — Log habit/habit-streak activity
- **debriefbus** — Generate post-context debriefs
- **ingestbus/extractor** — Claude Code extractor used by toolClassifyTasks to classify unlinked tasks by context

## Notes

- **No database operations** — MCP is a pure dispatcher; all actual work is delegated to business layer
- **Background processing** — `toolClassifyTasks()` launches a goroutine using `context.Background()` (not the request context) so AI extraction continues after HTTP response returns; returns only `unlinkedCount` and a status message
- **Typed clarification options** — `ContextAssignmentOptions`, `NewContextOptions`, `AmbiguousActionOptions`, `AmbiguousDeadlineOptions` in `clarificationbus/options.go` replace raw `map[string]any` at all `NewClarificationItem.AnswerOptions` creation sites; TypeScript counterparts are codegen'd via `tygo`
- **API key auth** — All MCP routes require `X-API-Key` header; validated by middleware in `route.go`
- **Synchronous tool calls** — Most tools wait for business layer result; only context classification is async
- **Error wrapping** — Tool handlers catch errors and return as toolResult with IsError=true, preserving JSON-RPC envelope integrity
