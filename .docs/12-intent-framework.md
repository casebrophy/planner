# Intent framework

The intent framework takes high-level goals, decomposes into executable plans, and executes through pluggable adapters. It is the system's agentic layer — everything else captures what has happened; this does something in the world.

**Core principle:** Present before executing — always generate plan + wait for confirmation. No exceptions. The confirmation gate is a first-class architectural component.

---

## Three-Tier Adapters

**Tier 1 — Declarative:** Defined entirely as structured data (intent name, slot schema, fill rules, execution spec, confirmation template). No code required. Added via conversation or Automations UI.

**Tier 2 — Claude-generated:** When an unrecognised intent is expressed, Claude proposes a new Tier 1 adapter definition through a conversational creation flow. If confirmed, saved to registry as Tier 1. No code, no deployment. Primary discoverability mechanism.

**Tier 3 — Code-backed:** Registered adapters with a custom Go implementation. Reserved for complex workflows that cannot be expressed declaratively (multi-step auth, stateful interactions, SDK-level integrations). Added via deployment.

---

## Intent Lifecycle

1. **Recognition** — check registry for matching adapter; if no match, offer adapter creation or fall back to task
2. **Slot filling** — fill from context, transaction history, conversation, preferences; ask for any remaining required slots
3. **Plan generation** — Claude synthesises filled slots into a specific, complete, human-readable plan
4. **Confirmation gate** — user confirms, adjusts (regenerate), or cancels (optionally convert to task)
5. **Execution** — adapter performs the external action
6. **Outcome capture** — result stored, slot values recorded, usage count incremented
7. **Crystallisation check** — if confirmed N times consistently, promote stable slots to defaults

---

## Slot Schema

```go
type Slot struct {
    Name        string       // e.g. "dish", "store", "party_size"
    Description string       // shown to user when asking
    Type        SlotType     // string | integer | date | datetime | boolean | enum | list
    Required    bool
    FillStrategy []FillRule  // ordered list of strategies to try
    Default     *string      // pre-filled default if all strategies fail and not required
}

type FillRule struct {
    Strategy string   // "context_pattern" | "transaction_history" | "user_preference"
                      // | "ask" | "recent_activity" | "calendar"
    Config   string   // JSON config for the strategy
}
```

---

## Fill Strategies

- **`context_pattern`** — look for slot value in the active context engine
- **`transaction_history`** — infer from past transactions (e.g. preferred store from purchase history)
- **`user_preference`** — look in explicitly saved user preferences
- **`recent_activity`** — look in recent conversation/activity window
- **`calendar`** — infer from upcoming calendar events
- **`ask`** — ask the user; always last resort; Claude asks in natural language, not as a form field

Strategies are tried in order; first confident value wins. Required slots that fail all strategies fall through to `ask`.

---

## Adapter Creation Flow

1. **Confirm the intent** — Claude restates what it understood and asks if the user wants it handled automatically going forward
2. **Name and describe** — Claude proposes a name and trigger phrases; user confirms or adjusts
3. **Walk through slots** — Claude identifies required information, proposes fill strategies for each, user confirms or changes
4. **Define execution** — Claude explains what the adapter will actually do and asks user to confirm it's correct
5. **Review complete definition** — Claude presents the full adapter in plain language (not JSON) and asks for final confirmation
6. **Save and test** — adapter saved to registry; Claude offers to run immediately or wait until next time

The creation conversation is stored as a `workflow_creation_sessions` record so adapter origin is always traceable.

---

## Crystallisation

- Triggers after **N=5** confirmed + executed runs (default)
- **Stable** = same or very similar value in at least 4 of 5 executions
- Stable slots become defaults — pre-filled in confirmation, no longer asked
- Only variable slots remain as questions; familiar flow becomes one question + one confirmation
- Crystallised Tier 2 adapters are candidates for promotion to Tier 3 (specification already fully defined)

---

## Data Model

```sql
CREATE TABLE intent_adapters (
    id               TEXT PRIMARY KEY,
    name             TEXT NOT NULL,
    description      TEXT NOT NULL,
    tier             INTEGER NOT NULL,        -- 1 | 2 | 3
    trigger_phrases  TEXT NOT NULL,           -- JSON array of example phrases
    slot_schema      TEXT NOT NULL,           -- JSON: Slot[] definition
    execution_spec   TEXT,                    -- JSON: for Tier 1/2 declarative execution
    handler_name     TEXT,                    -- for Tier 3: name of registered Go handler
    status           TEXT NOT NULL DEFAULT 'active',  -- active | draft | deprecated
    usage_count      INTEGER NOT NULL DEFAULT 0,
    crystallised     INTEGER NOT NULL DEFAULT 0,
    crystallised_at  TEXT,
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
);

CREATE TABLE intent_executions (
    id           TEXT PRIMARY KEY,
    adapter_id   TEXT NOT NULL REFERENCES intent_adapters(id),
    slot_values  TEXT NOT NULL,               -- JSON: actual values used
    plan         TEXT NOT NULL,               -- the plan shown to the user
    outcome      TEXT NOT NULL,               -- success | failure | cancelled
    result       TEXT,                        -- JSON: execution result or error
    created_at   TEXT NOT NULL
);

CREATE TABLE workflow_creation_sessions (
    id           TEXT PRIMARY KEY,
    adapter_id   TEXT REFERENCES intent_adapters(id),  -- null until saved
    transcript   TEXT NOT NULL,               -- full creation conversation
    completed    INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT NOT NULL
);
```

---

## MCP Tools

- **`recognise_intent`** — given natural language input, returns best matching adapter with confidence or null
- **`get_adapter`** — returns full adapter definition including slot schema and fill strategies
- **`list_adapters`** — returns all active adapters with names, descriptions, usage counts, crystallisation status
- **`fill_slots`** — given adapter + current context, attempts auto-fill; returns filled slots with provenance and unfilled list
- **`execute_intent`** — executes a confirmed plan; called only after user confirmation; records to `intent_executions`
- **`save_adapter`** — saves new or updated adapter definition; called at end of creation flow
