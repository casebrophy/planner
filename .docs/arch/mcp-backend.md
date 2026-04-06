# MCP Backend System

> JSON-RPC 2.0 Model Context Protocol server that exposes task, context, email, event, clarification, thread, and observation management as MCP tools. Acts as a facade over eight business domains — no business logic of its own, purely translates MCP tool calls into business layer operations. The `classify_tasks` tool uses `clarificationbus.ContextAssignmentOptions` (typed struct) for clarification `AnswerOptions` JSON rather than a raw `map[string]any`.

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
    timeBlockBus     *timeblockbus.Business
    clarificationBus *clarificationbus.Business
    threadBus        *threadbus.Business
    observationBus   *observationbus.Business
    debriefBus       *debriefbus.Business
    dailyPlanBus     *dailyplanbus.Business
    noteBus          *notebus.Business
    tagBus           *tagbus.Business
    activityLogBus   *activitylogbus.Business
}
```

## File Map

### App (Handlers)
- `app/domain/mcpapp/mcpapp.go` — **handle()** — POST /mcp, JSON-RPC dispatcher (initialize, tools/list, tools/call). **callTool()** — routes tool name to handler (37 cases). Task tools: **toolCreateTask()**, **toolListTasks()**, **toolGetTask()**, **toolUpdateTask()**, **toolCompleteTask()** (complete_task and update_task fire debriefBus.OnTaskCompleted). Context tools: **toolCreateContext()**, **toolGetContext()**, **toolListContexts()**, **toolUpdateContext()** (status → closed fires debriefBus.OnContextClosed). Email tools: **toolListEmails()**, **toolGetEmail()**. Event tools: **toolCreateEvent()**, **toolListEvents()**, **toolGetEvent()**, **toolUpdateEvent()**, **toolDeleteEvent()**. Clarification tools: **toolGetClarificationQueue()**, **toolResolveClarification()**, **toolSnoozeClarification()**. Thread tools: **toolAddThreadEntry()**, **toolGetThread()**. Observation tools: **toolRecordOutcome()**, **toolGetOutcomeObservations()**. Daily plan/schedule tools: **toolGetDailyPlan()**, **toolGenerateDailyPlan()**, **toolGetSchedule()**, **toolCreateTimeBlock()**, **toolConfirmTimeBlock()**. Dependency tools: **toolAddTaskDependency()**, **toolRemoveTaskDependency()**, **toolGetTaskDependencies()**. Note tools: **toolCreateNote()**, **toolSearchNotes()**, **toolListNotesByTag()**. Activity/tag tools: **toolLogActivity()**, **toolGetStreaks()**, **toolClassifyTasks()**.
- `app/domain/mcpapp/model.go` — JSON-RPC 2.0 request/response types, MCP protocol types
- `app/domain/mcpapp/tools.go` — Tool definitions registry (`var tools []toolDef`) with schemas for all 37 MCP tools
- `app/domain/mcpapp/route.go` — Route registration, wires up all business domains via their stores

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

### ⚠ clarificationbus.ContextAssignmentOptions (business/domain/clarificationbus/options.go)
`toolClassifyTasks()` marshals this typed struct into `AnswerOptions` JSON for low-confidence task-context matches (inside background goroutine). Changing field names or types affects:
- `mcpapp.go` — `toolClassifyTasks()` builds `ContextAssignmentOptions{SuggestedContext, Confidence, AvailableContexts}` and converts `[]extractor.ContextRef` → `[]clarificationbus.ContextRef` before marshaling
- Frontend `ClarificationCard` component — deserializes `answer_options` JSON for `context_assignment` kind; field renames break the UI
- `app/domain/classifyapp/classifyapp.go` — same struct used for same purpose in the HTTP classify endpoint
- `business/domain/ingestbus/ingestbus.go` — same struct used in both email and text ingestion paths

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

### ⚠ notebus (business/domain/notebus/)
If NewNote struct or Note/QueryFilter changes:
- `mcpapp.go` — `toolCreateNote()`, `toolSearchNotes()`, `toolListNotesByTag()` must be updated
- `tools.go` — tool input schemas must be updated

### ⚠ tagbus (business/domain/tagbus/)
If Tag struct or methods change:
- `mcpapp.go` — no direct MCP tools for tag CRUD, but note tagging uses tagBus
- `tools.go` — no schema changes (tags exposed only through note tagging)

### ⚠ activitylogbus (business/domain/activitylogbus/)
If ActivityLog struct or methods change:
- `mcpapp.go` — `toolLogActivity()` and `toolGetStreaks()` must be updated
- `tools.go` — tool input schemas must be updated

### ⚠ timeblockbus (business/domain/timeblockbus/)
If TimeBlock or UpdateTimeBlock structs change:
- `mcpapp.go` — `toolCreateTimeBlock()` and `toolConfirmTimeBlock()` must be updated
- `tools.go` — tool input schemas must be updated

### ⚠ taskbus dependencies (business/domain/taskbus/)
If dependency operations change:
- `mcpapp.go` — `toolAddTaskDependency()`, `toolRemoveTaskDependency()`, `toolGetTaskDependencies()` must be updated
- `tools.go` — tool input schemas must be updated

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

### Task Dependency Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| add_task_dependency | Add dependency between tasks | task_id, depends_on_id |
| remove_task_dependency | Remove dependency between tasks | task_id, depends_on_id |
| get_task_dependencies | Get task upstream and downstream deps | task_id |

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

### Time Block & Schedule Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| get_schedule | Get merged events/time blocks for date range | (none) |
| create_time_block | Schedule task into time slot | task_id, starts_at, ends_at |
| confirm_time_block | Confirm proposed time block | block_id |

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
| get_inference_context | Get pre-assembled context for inference | use_case |

### Note Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| create_note | Create a new note | content |
| search_notes | Search notes by keyword/tag/context | (none) |
| list_notes_by_tag | List notes with a specific tag | tag_name |

### Activity & Tag Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| log_activity | Log activity for task or note | subject_type, subject_id |
| get_streaks | Get streak/frequency info for item | subject_type, subject_id |

### Batch Tools
| Tool | Description | Required Args |
|------|-------------|---------------|
| classify_tasks | Auto-classify unlinked tasks | (none) |

## Cross-Domain Dependencies

### Business Layer Buses
- **taskbus** — task CRUD operations (Create, Query, QueryByID, Update, Count, dependency ops)
- **contextbus** — context CRUD operations (Create, Query, QueryByID, Update)
- **emailbus** — email read operations (Query, QueryByID)
- **eventbus** — event CRUD operations (Create, Query, QueryByID, Update, Delete, Count)
- **timeblockbus** — time block operations (Create, Get, Confirm)
- **clarificationbus** — clarification queue operations (Query, Count, QueryByID, Resolve, Snooze)
- **threadbus** — thread operations (Create, Query)
- **observationbus** — observation operations (Create, Query)
- **debriefbus** — debrief workflows (OnTaskCompleted, OnContextClosed fired from task/context handlers)
- **dailyplanbus** — daily plan operations (Get, Generate)
- **notebus** — note CRUD operations (Create, Query, Search)
- **tagbus** — tag CRUD and association operations (used by notes)
- **activitylogbus** — activity log operations (Create, GetStreaks)

### Store Instances
All stores instantiated in `route.go`:
- `taskdb.NewStore()`, `taskdb.NewDependencyStore()`
- `contextdb.NewStore()`
- `emaildb.NewStore()`
- `eventdb.NewStore()`
- `timeblockdb.NewStore()`
- `clarificationdb.NewStore()`
- `threaddb.NewStore()`
- `observationdb.NewStore()`
- `dailyplandb.NewStore()`
- `notedb.NewStore()`
- `tagdb.New()`
- `activitylogdb.NewStore()`

### Infrastructure
- **mid.Auth** — API key authentication middleware
- **web.Encoder** — rpcResponse implements this interface for HTTP response encoding
- **sqldb.ErrDBNotFound** — used for 404 handling in get operations
- **page** — pagination for list operations
- **order** — ordering/sorting for list operations

### Enum Types Used
- `taskstatus` — task workflow states (open, blocked, done, dismissed)
- `taskpriority` — task priority levels (low, medium, high, urgent)
- `taskenergy` — mental effort levels (low, medium, high)
- `contextkind` — context type (project, area)
- `clarificationkind` — clarification category (context_assignment, stale_task, etc.)
- `clarificationstatus` — clarification workflow (pending, snoozed, resolved, dismissed)
- `threadentrykind` — thread entry type (update, blocker, decision, etc.)
- `threadsource` — entry source (user, voice, email, transaction, system, claude)
- `observationkind` — observation type (duration_accuracy, blocker_profile, lesson, etc.)
