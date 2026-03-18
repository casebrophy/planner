# MCP Backend System

> JSON-RPC 2.0 Model Context Protocol server that exposes task and context management as MCP tools. Acts as a facade over `taskbus` and `contextbus` — no business logic of its own, purely translates MCP tool calls into business layer operations.

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

## File Map

### App (Handlers)
- `app/domain/mcpapp/mcpapp.go` — **handle()** — POST /mcp, JSON-RPC dispatcher (initialize, tools/list, tools/call). **callTool()** — routes tool name to handler. **toolCreateTask()**, **toolListTasks()**, **toolGetTask()**, **toolUpdateTask()**, **toolCompleteTask()** — task tools. **toolCreateContext()**, **toolGetContext()**, **toolListContexts()**, **toolUpdateContext()** — context tools.
- `app/domain/mcpapp/model.go` — JSON-RPC 2.0 request/response types, MCP protocol types
- `app/domain/mcpapp/tools.go` — Tool definitions registry (`var tools []toolDef`) with schemas for all 9 MCP tools
- `app/domain/mcpapp/route.go` — Route registration, wires up `taskbus` and `contextbus` via their stores

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

### ⚠ rpcResponse (app/domain/mcpapp/model.go)
Implements `web.Encoder` via `Encode()`. All handler methods return this type. Changes affect the entire MCP response format.

## Routes

| Method | Path | Handler | Auth |
|--------|------|---------|------|
| POST | /mcp | handle | API key (`mid.Auth`) |

## MCP Tools Exposed

| Tool | Description | Required Args |
|------|-------------|---------------|
| create_task | Create a new task | title |
| list_tasks | List tasks with filters | (none) |
| get_task | Get task by ID | task_id |
| update_task | Update task fields | task_id |
| complete_task | Mark task done | task_id |
| create_context | Create a new context | title |
| get_context | Get context + its tasks | context_id |
| list_contexts | List contexts by status | (none) |
| update_context | Update context fields | context_id |

## Cross-Domain Dependencies

- **taskbus** — used for all task CRUD operations (Create, Query, QueryByID, Update, Count)
- **contextbus** — used for all context CRUD operations (Create, Query, QueryByID, Update)
- **taskdb / contextdb** — instantiated in route.go to build business instances
- **mid.Auth** — API key authentication middleware
- **web.Encoder** — rpcResponse implements this interface for HTTP response encoding
- **sqldb.ErrDBNotFound** — used for 404 handling in get operations
- **page** — pagination for list operations
