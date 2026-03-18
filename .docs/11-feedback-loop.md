# Feedback loop, task threads & inactivity detection

Three connected features: **task threads** (running narrative on tasks/contexts), **inactivity detection** (surfaces a clarification card when something goes quiet), and **feedback loop** (debrief on completion feeds a structured observations store for pattern recognition).

---

## Task Threads

### Thread entry kinds

- `update` — general progress note from the user
- `blocker` — explicit statement that something is blocked
- `blocker_resolved` — a blocker was cleared
- `milestone` — a significant step completed
- `scope_change` — the task or situation changed in scope
- `timeline_slip` — a deadline or expected completion shifted
- `external_dep` — waiting on someone or something outside your control
- `decision` — a decision was made
- `observation` — system-inferred note (not from user directly)
- `email` — linked from an email extraction
- `transaction` — linked from a transaction match

`kind` is set by the system during extraction, never by the user manually.

### Thread extraction schema

```json
{
  "kind": "timeline_slip",
  "secondary_kinds": ["external_dep"],
  "summary": "Contractor pushed timeline by one week",
  "sentiment": "negative",
  "blocking_party": "contractor",
  "timeline_delta_days": 7,
  "requires_action": false,
  "action_description": null,
  "confidence": 0.88
}
```

`blocking_party` powers "waiting on X" pattern recognition. `requires_action` flags updates that imply something needs to happen and generates a clarification card or task automatically.

### Thread UI

- **Mobile:** bottom-sheet from task/context detail; scrollable entry list newest-first, text input at bottom; supports dictation
- **Web:** right-side panel in task/context detail; same content plus expandable entries showing full extracted metadata

---

## Inactivity Detection

### Clock resets / does NOT reset

Resets: thread entry added (any source), related email/transaction linked, status changes, linked task created or completed, clarification card answered.

Does NOT reset: item viewed in the app, clarification card snoozed, unrelated activity in the same context.

### Threshold table

| Priority | Default threshold |
|----------|------------------|
| urgent   | 1 day            |
| high     | 2 days           |
| medium   | 5 days           |
| low      | 14 days          |

Threshold can be overridden per-task via `expected_update_days`. Phase 8 ML cluster adjustment applies a longer/shorter threshold based on task similarity to known patterns.

### Inactivity card fields

- Subject task/context name and status
- Days since last update and summary of last update
- Quick-action buttons: Still in progress / Blocked / Completed / Deprioritised
- Free text input (goes through thread extraction)
- "Blocked" expands inline to capture `blocking_party` → stored as `blocker` entry
- Snooze option ("Remind me in 2 days")

Every inactivity event is stored regardless of response (Layer 1 pattern signal: frequency, correlation with cancellation/stall).

---

## Feedback Loop

### Debrief trigger rules

| Condition | Question asked |
|-----------|---------------|
| Completed on time, no blockers | "Anything worth noting for next time?" |
| Thread contains blocker entries | "What finally unblocked it?" (with suggested options) |
| Actual duration >2x estimate | "What caused the overrun?" (with suggested options) |
| Context closed | 3–4 card closing review sequence (outcome, biggest challenge, lesson, cost if financial data exists) |

### Adaptive framing rules

- Task debrief: single card, high priority, generated on move to `done`
- Context debrief: 3–4 card sequence shown together as "closing review"; skippable
- Card 2 of context debrief identifies the most prominent thread pattern and asks about it specifically
- Card 3 free text becomes a `lesson` observation with high weight in situational matching
- Card 4 fires only when financial data (linked transactions) exists

### Observation kinds

- `duration_accuracy` — estimated vs. actual days, ratio, task/context type
- `blocker_profile` — blocker count, blocking parties, resolution methods, blocked days
- `timeline_profile` — actual months, slip count, primary slip cause, most blocking party
- `lesson` — free-text insight derived from thread analysis or explicit user input
- `completion_pattern` — how the task/context resolved
- `scope_change` — scope delta and cause
- `cost_profile` — expected vs. actual cost, variance

Low-confidence inferences (<0.6) are eligible for clarification cards. High-confidence stored silently. User-provided observations are `confidence: 1.0` ground truth; inferred observations carry computed score.

---

## Data Model

### thread_entries

```sql
CREATE TABLE thread_entries (
    id              TEXT PRIMARY KEY,
    subject_type    TEXT NOT NULL,     -- task | context
    subject_id      TEXT NOT NULL,
    kind            TEXT NOT NULL,     -- see kinds above
    content         TEXT NOT NULL,     -- human-readable entry text
    metadata        TEXT,              -- JSON: extracted fields
    source          TEXT NOT NULL,     -- user | voice | email | transaction | system | claude
    source_id       TEXT,              -- FK to originating record if applicable
    sentiment       TEXT,              -- positive | neutral | negative | mixed
    requires_action INTEGER DEFAULT 0,
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_thread_subject ON thread_entries(subject_type, subject_id, created_at DESC);
CREATE INDEX idx_thread_kind    ON thread_entries(kind, created_at DESC);
CREATE INDEX idx_thread_action  ON thread_entries(requires_action)
    WHERE requires_action = 1;
```

### inactivity_checks

```sql
CREATE TABLE inactivity_checks (
    id                  TEXT PRIMARY KEY,
    subject_type        TEXT NOT NULL,
    subject_id          TEXT NOT NULL UNIQUE,
    last_activity_at    TEXT NOT NULL,
    threshold_days      REAL NOT NULL,
    last_triggered_at   TEXT,
    clarification_id    TEXT REFERENCES clarification_items(id),
    updated_at          TEXT NOT NULL
);
```

### outcome_observations

```sql
CREATE TABLE outcome_observations (
    id              TEXT PRIMARY KEY,
    subject_type    TEXT NOT NULL,     -- task | context
    subject_id      TEXT NOT NULL,
    kind            TEXT NOT NULL,     -- duration_accuracy | blocker_profile | cost_profile
                                       -- timeline_profile | relationship_profile | lesson
                                       -- completion_pattern | scope_change
    data            TEXT NOT NULL,     -- JSON: structured observation
    source          TEXT NOT NULL,     -- user | inferred
    confidence      REAL DEFAULT 1.0,
    weight          REAL DEFAULT 1.0,  -- lesson observations get higher weight
    created_at      TEXT NOT NULL
);

CREATE INDEX idx_observations_subject ON outcome_observations(subject_type, subject_id);
CREATE INDEX idx_observations_kind    ON outcome_observations(kind, created_at DESC);
```

### tasks — additions

```sql
ALTER TABLE tasks ADD COLUMN expected_update_days REAL;  -- null = inferred
ALTER TABLE tasks ADD COLUMN last_thread_at TEXT;        -- denormalised for inactivity queries
ALTER TABLE tasks ADD COLUMN debrief_status TEXT DEFAULT 'pending';  -- pending | done | skipped
```

### contexts — additions

```sql
ALTER TABLE contexts ADD COLUMN last_thread_at TEXT;
ALTER TABLE contexts ADD COLUMN debrief_status TEXT DEFAULT 'pending';
ALTER TABLE contexts ADD COLUMN outcome TEXT;  -- went_well | mixed | difficult | ongoing_issues
```

---

## MCP Tools

- `add_thread_entry` — adds an update to a task or context thread; triggers extraction before storage
- `get_thread` — returns the full thread for a task or context; used for narrative history and debrief generation
- `get_outcome_observations` — returns stored observations for a subject or subject type; used in situational matching
- `record_outcome` — stores an outcome observation after a debrief card is answered
