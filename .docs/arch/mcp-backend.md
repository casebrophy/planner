# MCP Backend System

> JSON-RPC 2.0 Model Context Protocol server that exposes task, context, email, event, clarification, thread, and observation management as MCP tools. Acts as a facade over eight business domains — no business logic of its own, purely translates MCP tool calls into business layer operations.

## Core Types

```go
// app/domain/mcpapp/model.go

type rpcRequest struct {
    JSONRPC string          `json:"jsonrpc"`
    ID      any             `json:"id"`
    Method  string          `json:"method"`
    Params  json.RawMessage `json:"params,omitempty"`
}

type rpcResponse struct {
    JSONRPC string    `json:"jsonrpc"`
    ID      any       `json:"id"`
    Result  any       `json:"result,omitempty"`
    Error   *rpcError `json:"error,omitempty"`
}

type rpcError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type toolDef struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    InputSchema any    `json:"inputSchema"`
}

type toolCallParams struct {
    Name      string          `json:"name"`
    Arguments json.RawMessage `json:"arguments,omitempty"`
}

type initializeResult struct {
    ProtocolVersion string     `json:"protocolVersion"`
    ServerInfo      serverInfo `json:"serverInfo"`
    Capabilities    any        `json:"capabilities"`
}

type serverInfo struct {
    Name    string `json:"name"`
    Version string `json:"version"`
}

type toolResult struct {
    Content []toolContent `json:"content"`
    IsError bool          `json:"isError,omitempty"`
}

type toolContent struct {
    Type string `json:"type"`
    Text string `json:"text"`
}
```

```go
// app/domain/mcpapp/mcpapp.go

type app struct {
    taskBus          *taskbus.Business
    contextBus       *contextbus.Business
    emailBus         *emailbus.Business
    eventBus         *eventbus.Business
    clarificationBus *clarificationbus.Business
    threadBus        *threadbus.Business
    observationBus   *observationbus.Business
    debriefBus       *debriefbus.Business
    dailyPlanBus     *dailyplanbus.Business
}
```

## File Map

### App (Handlers)
- `app/domain/mcpapp/mcpapp.go` — **handle()** — POST /mcp, JSON-RPC dispatcher (initialize, tools/list, tools/call). **callTool()** — routes tool name to handler (25 cases). **toolCreateTask()**, **toolListTasks()**, **toolGetTask()**, **toolUpdateTask()**, **toolCompleteTask()** — task tools (complete_task and update_task fire debriefBus.OnTaskCompleted in a goroutine). **toolCreateContext()**, **toolGetContext()**, **toolListContexts()**, **toolUpdateContext()** — context tools (update_context fires debriefBus.OnContextClosed in a goroutine when status → closed). **toolListEmails()**, **toolGetEmail()** — email tools. **toolCreateEvent()**, **toolListEvents()**, **toolGetEvent()**, **toolUpdateEvent()**, **toolDeleteEvent()** — event tools (full CRUD). **toolGetClarificationQueue()**, **toolResolveClarification()**, **toolSnoozeClarification()** — clarification tools. **toolAddThreadEntry()**, **toolGetThread()** — thread tools. **toolRecordOutcome()**, **toolGetOutcomeObservations()** — observation tools. **toolGetDailyPlan()**, **toolGenerateDailyPlan()** — daily plan tools.
- `app/domain/mcpapp/model.go` — JSON-RPC 2.0 request/response types, MCP protocol types
- `app/domain/mcpapp/tools.go` — Tool definitions registry (`var tools []toolDef`) with schemas for all 25 MCP tools
- `app/domain/mcpapp/route.go` — Route registration, wires up `taskbus`, `contextbus`, `emailbus`, `eventbus`, `clarificationbus`, `threadbus`, `observationbus`, `debriefbus`, `dailyplanbus` via their stores

## Impact Callouts

### ⚠ toolDef registry (app/domain/mcpapp/tools.go)
Adding a new MCP tool requires:
- `tools.go` — add `toolDef` entry with name, description, inputSchema
- `mcpapp.go` — add `case` in `callTool()` switch, implement `tool{Name}()` method

### ⚠ taskbus.NewTask / taskbus.UpdateTask (business/domain/taskbus/)
If these structs gain new fields:
- `mcpapp.go` — `toolCreateTask()` and `toolUpdateTask()` must parse/pass the new field
- `tools.go` — tool input schemas must be updated to expose the new field

### ⚠ contextbus.NewContext / contextbus.UpdateContext (business/domain/contextbus/)
If these structs gain new fields:
- `mcpapp.go` — `toolCreateContext()` and `toolUpdateContext()` must parse/pass the new field
- `tools.go` — tool input schemas must be updated

### ⚠ emailbus (business/domain/emailbus/)
If Email struct or QueryFilter changes:
- `mcpapp.go` — `toolListEmails()` and `toolGetEmail()` must handle new fields/filters
- `tools.go` — tool input schemas must be updated

### ⚠ clarificationbus (business/domain/clarificationbus/)
If ClarificationItem struct, QueryFilter, or Resolve/Snooze methods change:
- `mcpapp.go` — `toolGetClarificationQueue()`, `toolResolveClarification()`, `toolSnoozeClarification()` must be updated
- `tools.go` — tool input schemas must be updated

### ⚠ threadbus (business/domain/threadbus/)
If NewThreadEntry struct or ThreadEntry changes:
- `mcpapp.go` — `toolAddThreadEntry()` and `toolGetThread()` must handle new fields
- `tools.go` — tool input schemas must be updated (especially kind and source enums)

### ⚠ observationbus (business/domain/observationbus/)
If NewObservation struct changes:
- `mcpapp.go` — `toolRecordOutcome()` must handle new fields
- `tools.go` — tool input schemas must be updated (especially kind enum)

### ⚠ eventbus (business/domain/eventbus/)
If NewEvent or UpdateEvent structs change:
- `mcpapp.go` — `toolCreateEvent()` and `toolUpdateEvent()` must parse/pass new fields
- `tools.go` — tool input schemas must be updated
- `toolDeleteEvent()` must handle any new deletion side effects

### ⚠ rpcResponse (app/domain/mcpapp/model.go)
Implements `web.Encoder` via `Encode()`. All handler methods return this type. Changes affect the entire MCP response format.

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | /mcp | handle | API key (`mid.Auth`) |

## MCP Tools Exposed

### Task Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| create_task | Create a new task | title |
| list_tasks | List tasks with filters | (none) |
| get_task | Get task by ID | task_id |
| update_task | Update task fields | task_id |
| complete_task | Mark task done | task_id |

### Context Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| create_context | Create a new context | title |
| get_context | Get context + its tasks | context_id |
| list_contexts | List contexts by status | (none) |
| update_context | Update context fields | context_id |

### Email Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| list_emails | List ingested emails with filters | (none) |
| get_email | Get full email details by ID | email_id |

### Event Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| create_event | Create a new event | title, starts_at, ends_at |
| list_events | List events with optional filters | (none) |
| get_event | Get event by ID | event_id |
| update_event | Update event fields | event_id |
| delete_event | Delete an event by ID | event_id |

### Clarification Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| get_clarification_queue | Get pending clarification items | (none) |
| resolve_clarification | Submit answer to a clarification | clarification_id, answer |
| snooze_clarification | Snooze a clarification for N hours | clarification_id |

### Thread Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| add_thread_entry | Add update to a task/context thread | subject_type, subject_id, kind, content |
| get_thread | Get full thread history | subject_type, subject_id |

### Observation Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| record_outcome | Record an outcome observation | subject_type, subject_id, kind, data |
| get_outcome_observations | Query observations for a task or context | subject_type, subject_id |

### Daily Plan Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| get_daily_plan | Get today's plan with grouped items | (none, optional date) |
| generate_daily_plan | Generate or regenerate a daily plan | (none, optional date) |

## Cross-Domain Dependencies

- **taskbus** — task CRUD operations (Create, Query, QueryByID, Update, Count)
- **contextbus** — context CRUD operations (Create, Query, QueryByID, Update)
- **emailbus** — email read operations (Query, QueryByID)
- **eventbus** — event CRUD operations (Create, Query, QueryByID, Update, Delete, Count)
- **clarificationbus** — clarification queue operations (Query, Count, QueryByID, Resolve, Snooze)
- **threadbus** — thread operations (Create, Query)
- **observationbus** — observation operations (Create)
- **debriefbus** — debrief workflows (OnTaskCompleted, OnContextClosed fired from task/context handlers)
- **dailyplanbus** — daily plan operations (Get, Generate)
- **taskdb / contextdb / emaildb / eventdb / clarificationdb / threaddb / observationdb / dailyplandb** — all instantiated in route.go
- **mid.Auth** — API key authentication middleware
- **web.Encoder** — rpcResponse implements this interface for HTTP response encoding
- **sqldb.ErrDBNotFound** — used for 404 handling in get operations
- **page** — pagination for list operations
- **Enum types used:** taskstatus, taskpriority, taskenergy, clarificationkind, clarificationstatus, threadentrykind, threadsource, observationkind
