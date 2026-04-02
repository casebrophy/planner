# Sidecar Orchestrator: Persistent Session with Observability

## Problem

The current `/inference` endpoint spawns a fresh `claude -p` process per request. Each invocation pays the full CLI startup cost (~2-3s), has no memory of prior requests, and provides no observability into token usage or context health.

## Solution

Replace the fire-and-forget `exec.Command` model with a **session-managed orchestrator**. A persistent Claude session (Opus) acts as a dispatcher: it receives inference requests, spawns subagents at the requested model level, and returns their output. The session persists across requests via `--resume`, rotating only when context is exhausted, on error, or on timeout.

Spawned agents get **read-only MCP access** to the planner API, enabling them to pull context they need rather than requiring callers to pre-bake everything into the prompt.

A new **`get_inference_context`** MCP tool provides pre-assembled context bundles for known use cases, reducing the number of tool calls agents need to make.

Full observability via new endpoints exposes token usage, session health, tool call patterns, and context growth trends.

---

## Architecture

```
HTTP POST /inference
    -> Go sidecar (mutex-serialized)
    -> claude -p "<request>" --resume <session_id> --model opus --output-format json
    -> Orchestrator (Opus, persistent session)
        -> Agent tool (model from request, read-only MCP tools)
        -> Agent queries planner via MCP (read-only)
        -> Agent returns result
    -> Parsed response -> HTTP response
```

### Session Lifecycle

1. **First request**: No session exists. Run `claude -p` with `--system-prompt` and the user's request. Capture `session_id` from JSON output. Store it in memory.
2. **Subsequent requests**: `--resume <session_id>` continues the conversation. The orchestrator remembers its system prompt and role.
3. **Rotation**: Discard session ID and start fresh on next request when any of:
   - Request exceeds 180s timeout
   - Claude CLI returns non-zero exit
   - Input tokens exceed context threshold (default 150k, configurable)
   - Manual rotation via `POST /inference/rotate`

### Request Serialization

A `sync.Mutex` ensures only one inference request runs at a time. Concurrent requests queue up. This prevents multiple claude processes competing for the same session.

---

## System Prompt

The orchestrator runs as Opus and receives this system prompt on session initialization:

```
You are the planner system's inference orchestrator. You run persistently on the server and handle automated inference requests from the planner's backend pipelines.

Your ONLY job is to dispatch work to subagents and return their raw results.

When you receive a request:
1. Read the "model" field to determine which model the subagent should use
2. Read the "prompt" field for the task to dispatch
3. If a "schema" field is present, include it in the agent's instructions as the required output format
4. Spawn a subagent using the Agent tool at the specified model
5. The subagent has read-only access to the planner's MCP tools (list_tasks, list_events, get_context, etc.) and a specialized get_inference_context tool for pre-assembled context bundles
6. Return ONLY the subagent's output text. No commentary, no wrapping, no additional formatting.

The requests come from automated pipelines:
- Daily plan generation: groups and prioritizes tasks for the day
- Email extraction: extracts action items, deadlines, and context from emails
- Text/voice extraction: extracts tasks and events from free-form input
- Thread classification: classifies thread entries by kind, sentiment, and urgency

Each request includes its own detailed prompt with classification rules and expected output schema. Trust the prompt. Do not second-guess the pipeline's instructions.

Never reason about the task yourself. Never add your own analysis. You are a dispatcher.
```

---

## Inference Request/Response

### Request (unchanged wire format)

```json
{
  "prompt": "string (required) - the full prompt for the subagent",
  "schema": "string (optional) - JSON schema for structured output",
  "model": "string (required, default 'haiku') - model for the subagent, NOT the orchestrator"
}
```

### Response (unchanged wire format)

```json
{
  "result": "string - the subagent's raw output",
  "model": "string - the model that was used"
}
```

The orchestrator is always Opus. The `model` field controls the subagent.

---

## MCP Integration

### Agent Tool Access

Spawned agents get access to **read-only** MCP tools from the planner API:

| Tool | Purpose |
|------|---------|
| `list_tasks` | Query tasks with filters |
| `get_task` | Get full task details |
| `list_contexts` | List active contexts |
| `get_context` | Get context with summary |
| `list_events` | Query events by date range |
| `get_event` | Get event details |
| `list_emails` | Query ingested emails |
| `get_email` | Get full email details |
| `get_thread` | Get thread history |
| `get_daily_plan` | Get current daily plan |
| `get_schedule` | Get merged schedule |
| `get_outcome_observations` | Query outcome data |
| `get_clarification_queue` | Get pending clarifications |
| `get_inference_context` | **New** - pre-assembled context bundle |

**Excluded (write) tools:** `create_task`, `update_task`, `complete_task`, `create_context`, `update_context`, `create_event`, `update_event`, `delete_event`, `add_thread_entry`, `record_outcome`, `resolve_clarification`, `snooze_clarification`, `generate_daily_plan`, `create_time_block`, `confirm_time_block`.

Write operations stay in the Go business logic where side effects are predictable.

### New MCP Tool: `get_inference_context`

A composite tool that returns pre-assembled context for known pipeline use cases, reducing multi-tool-call round trips.

```json
{
  "name": "get_inference_context",
  "description": "Get pre-assembled context for a specific inference use case. Returns all relevant data in a single call instead of requiring multiple tool calls.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "use_case": {
        "type": "string",
        "enum": ["daily_plan", "email_extraction", "text_extraction", "thread_classification"],
        "description": "The inference pipeline requesting context"
      },
      "date": {
        "type": "string",
        "description": "ISO 8601 date for date-scoped use cases (daily_plan, text_extraction)"
      },
      "subject_id": {
        "type": "string",
        "description": "UUID of the subject (for thread_classification)"
      },
      "subject_type": {
        "type": "string",
        "enum": ["task", "context"],
        "description": "Type of subject (for thread_classification)"
      }
    },
    "required": ["use_case"]
  }
}
```

**Returns by use case:**

| Use Case | Data Returned |
|----------|---------------|
| `daily_plan` | Open tasks (with context titles), today's events, yesterday's carryover items |
| `email_extraction` | Active contexts (id, title, description, keywords) for context matching |
| `text_extraction` | Active contexts + today's events for temporal grounding |
| `thread_classification` | Thread history (last 10 entries), subject details (task or context) |

This tool is implemented in the planner API's MCP handler alongside existing tools.

---

## Observability

### Session Manager State

```go
type SessionManager struct {
    mu           sync.Mutex
    sessionID    string
    createdAt    time.Time
    systemPrompt string
    contextMax   int            // default 150000 tokens
    timeout      time.Duration  // default 180s
    requests     []RequestMetric
    sessions     []SessionSummary
    toolCalls    []ToolCallMetric
}

type RequestMetric struct {
    ID           string    `json:"id"`
    Timestamp    time.Time `json:"timestamp"`
    InputTokens  int       `json:"input_tokens"`
    OutputTokens int       `json:"output_tokens"`
    DurationMs   int       `json:"duration_ms"`
    AgentModel   string    `json:"agent_model"`
    PromptPrefix string    `json:"prompt_prefix"` // first 100 chars
    Success      bool      `json:"success"`
    Error        string    `json:"error,omitempty"`
}

type SessionSummary struct {
    SessionID       string    `json:"session_id"`
    CreatedAt       time.Time `json:"created_at"`
    EndedAt         time.Time `json:"ended_at"`
    TotalRequests   int       `json:"total_requests"`
    EndReason       string    `json:"end_reason"` // "timeout", "error", "context_full", "manual"
    PeakInputTokens int      `json:"peak_input_tokens"`
}

type ToolCallMetric struct {
    Timestamp  time.Time `json:"timestamp"`
    SessionID  string    `json:"session_id"`
    RequestID  string    `json:"request_id"`
    ToolName   string    `json:"tool_name"`
    DurationMs int       `json:"duration_ms"`
}
```

### Observability Endpoints

#### `GET /inference/status` - Current session health

```json
{
  "session_id": "abc-123",
  "age_seconds": 3400,
  "total_requests": 12,
  "latest_input_tokens": 45000,
  "context_max": 150000,
  "context_usage_pct": 30.0,
  "token_growth": [1200, 3400, 8900, 15000, 23000, 45000],
  "avg_duration_ms": 4200,
  "duration_trend": [2100, 3000, 3800, 5200, 6400],
  "requests_since_rotation": 12
}
```

**Context pollution signal:** If `token_growth` is linear but `duration_trend` curves upward, the session is degrading. The frontend (or an alert) can flag this.

#### `GET /inference/history` - Past sessions

```json
{
  "sessions": [
    {
      "session_id": "abc-123",
      "created_at": "2026-04-02T08:00:00Z",
      "ended_at": "2026-04-02T12:15:00Z",
      "total_requests": 47,
      "peak_input_tokens": 148000,
      "end_reason": "context_full"
    }
  ]
}
```

#### `GET /inference/tools` - Tool call patterns

```json
{
  "tool_frequency": {
    "list_tasks": 47,
    "list_events": 43,
    "get_inference_context": 38,
    "get_context": 12
  },
  "common_sequences": [
    ["get_inference_context", "list_tasks"],
    ["get_inference_context"]
  ],
  "avg_calls_per_request": 2.1
}
```

When repeated multi-tool sequences appear that aren't covered by `get_inference_context`, that's the signal to build a new composite MCP tool.

#### `POST /inference/rotate` - Force session rotation

```json
{
  "reason": "manual"
}
```

Response:
```json
{
  "old_session_id": "abc-123",
  "requests_served": 47,
  "peak_input_tokens": 148000,
  "rotated": true
}
```

---

## Token Tracking

The `--output-format json` output from `claude -p` returns an array of event objects. The `result` event contains token usage:

```json
[
  {"type": "result", "result": "...", "input_tokens": 45000, "output_tokens": 800}
]
```

We parse `input_tokens` and `output_tokens` from the result event. If these fields aren't present (older CLI versions), we estimate based on prompt length (1 token ~= 4 chars).

Context usage percentage = `input_tokens / contextMax * 100`.

Auto-rotation triggers when `input_tokens >= contextMax`.

---

## File Changes

### Modified files

| File | Change |
|------|--------|
| `zarf/sidecar/main.go` | Add new routes, initialize SessionManager |
| `zarf/sidecar/handlers.go` | Rewrite `inference()`, add `inferenceStatus()`, `inferenceHistory()`, `inferenceTools()`, `inferenceRotate()` |
| `app/domain/mcpapp/tools.go` | Add `get_inference_context` tool definition |
| `app/domain/mcpapp/mcpapp.go` | Implement `get_inference_context` handler |
| `.docs/08-ai-model-layer.md` | Update to document orchestrator architecture |

### New files

| File | Purpose |
|------|---------|
| `zarf/sidecar/session.go` | SessionManager struct, session lifecycle, metrics tracking |

---

## Configuration

| Env Var | Default | Description |
|---------|---------|-------------|
| `SIDECAR_CONTEXT_MAX` | `150000` | Token threshold for auto-rotation |
| `SIDECAR_REQUEST_TIMEOUT` | `180s` | Per-request timeout |
| `SIDECAR_SYSTEM_PROMPT` | (embedded) | Override system prompt path |
| `PLANNER_MCP_URL` | `http://localhost:8080/mcp` | MCP endpoint for agent tools |

---

## What This Does NOT Change

- The `foundation/claudecli` package and its callers remain unchanged. The Go business logic still calls `client.RunJSON()` which hits `POST /inference` on the sidecar. The sidecar's internal implementation changes, but the API contract stays the same.
- Model escalation logic stays in `claudecli.Client.RunJSON()`. The sidecar handles one model per request; escalation is the caller's responsibility.
- Write operations remain in Go business logic. Agents cannot mutate planner state.

---

## Future Extensions

- **Tool call logging for MCP extension discovery**: When `GET /inference/tools` shows repeated multi-tool sequences not covered by `get_inference_context`, build new composite tools.
- **Context pollution alerting**: Expose a webhook or log line when `duration_trend` slope exceeds a threshold, suggesting rotation.
- **Session warm-up**: On rotation, pre-seed the new session with a lightweight context summary from the planner API.
