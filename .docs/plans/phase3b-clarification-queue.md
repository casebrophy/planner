# Phase 3b: Clarification Queue — Implementation Plan

## Summary

Persistent queue of questions the system cannot resolve automatically, surfaced for user review via REST API and MCP tools. Also delivers: task/context threads, inactivity detection, context debrief flow, outcome observations.

**Dependencies:** Phase 3 (email ingestion) for pipeline triggers. Queue CRUD can be built and tested independently.
**Core UX:** Swipeable review deck ordered by computed priority. Each card shows question, Claude's guess, context-sensitive options. Resolution updates the underlying record.

## Decisions

| Question | Decision |
|----------|----------|
| Thread domain | Separate `threadbus` (spans tasks + contexts) |
| Resolution dispatcher | Start in `clarificationapp`, extract if it grows |
| Inactivity trigger | Background goroutine in main.go, every 15 min |
| Context merge | Deferred — placeholder option acknowledges overlap, creates TODO. **NOTE: full merge logic (reassign tasks, events, threads) is a future phase item** |
| Priority scoring | Simplified: `age_hours * 0.4 + kind_weight * 0.6` (static weight per kind) |
| "By the way" summary | Deferred to Phase 5b (pattern recognition) |

## Database Migrations

### Version 1.07 — clarification_items

```sql
CREATE TABLE clarification_items (
    clarification_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    kind             TEXT        NOT NULL CHECK (kind IN (
        'context_assignment', 'stale_task', 'ambiguous_deadline',
        'new_context', 'overlapping_contexts', 'ambiguous_action',
        'voice_reference', 'inactivity_prompt', 'context_debrief'
    )),
    status           TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'snoozed', 'resolved', 'dismissed')),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context', 'email', 'raw_input')),
    subject_id       UUID        NOT NULL,
    question         TEXT        NOT NULL,
    claude_guess     JSONB,
    reasoning        TEXT,
    answer_options   JSONB       NOT NULL,
    answer           JSONB,
    priority_score   REAL        NOT NULL DEFAULT 0.0,
    snoozed_until    TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at      TIMESTAMPTZ,
    PRIMARY KEY (clarification_id)
);
CREATE INDEX idx_clarification_pending ON clarification_items(status, priority_score DESC) WHERE status = 'pending';
CREATE INDEX idx_clarification_snoozed ON clarification_items(snoozed_until) WHERE status = 'snoozed';
CREATE INDEX idx_clarification_subject ON clarification_items(subject_type, subject_id);
```

### Version 1.08 — thread_entries

```sql
CREATE TABLE thread_entries (
    entry_id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context')),
    subject_id       UUID        NOT NULL,
    kind             TEXT        NOT NULL CHECK (kind IN (
        'update', 'blocker', 'blocker_resolved', 'milestone',
        'scope_change', 'timeline_slip', 'external_dep',
        'decision', 'observation', 'email', 'transaction'
    )),
    content          TEXT        NOT NULL,
    metadata         JSONB,
    source           TEXT        NOT NULL DEFAULT 'user' CHECK (source IN ('user', 'voice', 'email', 'transaction', 'system', 'claude')),
    source_id        UUID,
    sentiment        TEXT        CHECK (sentiment IN ('positive', 'neutral', 'negative', 'mixed')),
    requires_action  BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (entry_id)
);
CREATE INDEX idx_thread_subject ON thread_entries(subject_type, subject_id, created_at DESC);
CREATE INDEX idx_thread_action ON thread_entries(requires_action) WHERE requires_action = TRUE;
```

### Version 1.09 — inactivity_checks

```sql
CREATE TABLE inactivity_checks (
    check_id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type         TEXT        NOT NULL CHECK (subject_type IN ('task', 'context')),
    subject_id           UUID        NOT NULL UNIQUE,
    last_activity_at     TIMESTAMPTZ NOT NULL,
    threshold_days       REAL        NOT NULL,
    last_triggered_at    TIMESTAMPTZ,
    clarification_id     UUID        REFERENCES clarification_items(clarification_id),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (check_id)
);
```

### Version 1.10 — outcome_observations

```sql
CREATE TABLE outcome_observations (
    observation_id   UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context')),
    subject_id       UUID        NOT NULL,
    kind             TEXT        NOT NULL CHECK (kind IN (
        'duration_accuracy', 'blocker_profile', 'timeline_profile',
        'lesson', 'completion_pattern', 'scope_change', 'cost_profile'
    )),
    data             JSONB       NOT NULL,
    source           TEXT        NOT NULL CHECK (source IN ('user', 'inferred')),
    confidence       REAL        NOT NULL DEFAULT 1.0,
    weight           REAL        NOT NULL DEFAULT 1.0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (observation_id)
);
CREATE INDEX idx_observations_subject ON outcome_observations(subject_type, subject_id);
CREATE INDEX idx_observations_kind ON outcome_observations(kind, created_at DESC);
```

### Version 1.11 — ALTER tasks and contexts

```sql
ALTER TABLE tasks ADD COLUMN expected_update_days REAL;
ALTER TABLE tasks ADD COLUMN last_thread_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN debrief_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (debrief_status IN ('pending', 'done', 'skipped'));

ALTER TABLE contexts ADD COLUMN last_thread_at TIMESTAMPTZ;
ALTER TABLE contexts ADD COLUMN debrief_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (debrief_status IN ('pending', 'done', 'skipped'));
ALTER TABLE contexts ADD COLUMN outcome TEXT
    CHECK (outcome IN ('went_well', 'mixed', 'difficult', 'ongoing_issues'));
```

## New Enum Types

- `clarificationkind/` — ContextAssignment, StaleTask, AmbiguousDeadline, NewContext, OverlappingContexts, AmbiguousAction, VoiceReference, InactivityPrompt, ContextDebrief
- `clarificationstatus/` — Pending, Snoozed, Resolved, Dismissed
- `threadentrykind/` — Update, Blocker, BlockerResolved, Milestone, ScopeChange, TimelineSlip, ExternalDep, Decision, Observation, Email, Transaction
- `threadsource/` — User, Voice, EmailSource, TransactionSource, System, Claude
- `debriefstatus/` — Pending, Done, Skipped
- `observationkind/` — DurationAccuracy, BlockerProfile, TimelineProfile, Lesson, CompletionPattern, ScopeChange, CostProfile

## New Go Packages

### clarificationbus

```
business/domain/clarificationbus/
  model.go            — ClarificationItem, NewClarificationItem, ResolveClarificationItem
  clarificationbus.go — Storer interface + Business (Create, Resolve, Snooze, Dismiss, Query, QueryByID, Count, UnsnoozeExpired)
  filter.go           — QueryFilter (Status, Kind, SubjectType, SubjectID)
  order.go            — OrderByPriorityScore (default DESC), OrderByCreatedAt
  stores/clarificationdb/ — standard store implementation
```

### threadbus

```
business/domain/threadbus/
  model.go       — ThreadEntry, NewThreadEntry
  threadbus.go   — Storer interface + Business (AddEntry, QueryBySubject, CountBySubject, QueryByID)
  filter.go      — QueryFilter (SubjectType, SubjectID, Kind, RequiresAction)
  order.go       — OrderByCreatedAt (default DESC)
  stores/threaddb/ — standard store implementation
```

### observationbus

```
business/domain/observationbus/
  model.go          — Observation, NewObservation
  observationbus.go — Storer interface + Business (Record, QueryBySubject, QueryByKind)
  filter.go         — QueryFilter (SubjectType, SubjectID, Kind)
  stores/observationdb/ — standard store implementation
```

## Clarification Lifecycle

```
pending ──resolve──> resolved
   │
   ├──snooze──> snoozed ──(timer expires)──> pending
   │
   └──dismiss──> dismissed
```

## Triggers

| Kind | Subject | When | Priority Weight |
|------|---------|------|-----------------|
| context_assignment | email/raw_input | Pipeline context match confidence < 0.7 | 0.7 |
| ambiguous_action | email | Pipeline can't determine if email contains task | 0.8 |
| new_context | context | Pipeline auto-creates context from email | 0.9 |
| inactivity_prompt | task | Inactivity job fires (no activity beyond threshold) | 0.6 |
| voice_reference | raw_input | MCP voice tool can't resolve reference | 0.7 |
| context_debrief | context | Context closed, 24h delay (pre-snoozed card) | 0.8 |
| ambiguous_deadline | task | Pipeline extracts deadline with low confidence | 0.5 |
| overlapping_contexts | context | Context engine detects similarity above threshold | 0.6 |

**Score formula:** `age_hours * 0.4 + kind_weight * 0.6`

## Inactivity Detection

Background goroutine, every 15 minutes. Checks tasks with status IN (todo, in_progress) where `last_activity_at + threshold_days < NOW()`.

Default thresholds: urgent=1d, high=2d, medium=5d, low=14d. Overridden by `tasks.expected_update_days` if set.

Activity resets on: thread entry added, task status change, linked clarification resolved.

## Context Debrief Flow

When context status → closed:
1. Set `debrief_status = 'pending'`
2. Create 3-4 clarification cards (kind=context_debrief), all pre-snoozed 24h
3. UnsnoozeExpired job flips them to pending after 24h

**Cards:**
1. Outcome: "How did [title] go?" → went_well / mixed / difficult / ongoing_issues
2. Biggest challenge: auto-generated from thread analysis
3. Lesson: free text "Anything worth knowing for next time?"
4. Cost (conditional): only if linked transactions exist

Resolution creates outcome_observations. After all cards resolved: `debrief_status = 'done'`.

## Resolution Dispatcher (in clarificationapp)

Maps kind + answer → side-effect:

| Kind | Action |
|------|--------|
| context_assignment | Update email/raw_input context_id |
| ambiguous_action | Create task or mark as no-task |
| new_context | No-op / edit / **merge deferred (TODO)** |
| inactivity_prompt | Thread update / complete / block / deprioritize |
| ambiguous_deadline | Update task due_date |
| context_debrief | Create outcome_observation, update context |
| overlapping_contexts | **Merge deferred (TODO)** / dismiss |

## API Endpoints

### Clarifications
| Method | Path | Handler |
|--------|------|---------|
| GET | `/api/v1/clarifications` | queryQueue (default: status=pending, order by priority DESC) |
| GET | `/api/v1/clarifications/{id}` | queryByID |
| POST | `/api/v1/clarifications/{id}/resolve` | resolve |
| POST | `/api/v1/clarifications/{id}/snooze` | snooze |
| POST | `/api/v1/clarifications/{id}/dismiss` | dismiss |
| GET | `/api/v1/clarifications/count` | countPending |

### Threads
| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/v1/threads` | addEntry |
| GET | `/api/v1/threads/{subject_type}/{subject_id}` | queryThread |

### Observations
| Method | Path | Handler |
|--------|------|---------|
| POST | `/api/v1/observations` | record |
| GET | `/api/v1/observations/{subject_type}/{subject_id}` | queryBySubject |

## MCP Tools

- `get_clarification_queue` — filter by status, paginated
- `resolve_clarification` — submit answer, triggers resolution dispatcher
- `snooze_clarification` — snooze for N hours (default 24)
- `add_thread_entry` — add update to task/context thread
- `get_thread` — full thread history for a subject
- `record_outcome` — store outcome observation

## Implementation Order

1. Enum types (6 packages)
2. Database migrations (1.07–1.11)
3. Thread domain (threadbus + threaddb + threadapp)
4. Observation domain (observationbus + observationdb + observationapp)
5. Clarification domain (clarificationbus + clarificationdb + clarificationapp)
6. MCP tools (wire all three domains into mcpapp)
7. ALTER tasks/contexts + update existing models/stores/handlers (~15 files)
8. Resolution dispatcher (in clarificationapp)
9. Inactivity detection (goroutine in main.go)
10. Context debrief flow (trigger on context close, card sequence)
11. Unsnooze job (periodic goroutine, every 5 min)
12. Arch docs (clarification-backend.md, thread-backend.md)
