# Data Model

Three top-level concepts: **contexts** (ongoing situations), **tasks** (discrete actions), **sources** (external data).

Database: **PostgreSQL** (via Docker, mapped to port 5433 locally).

## Entity Relationships

- contexts → tasks (one-to-many, optional), context_events (timeline), emails, tags (many-to-many), thread_entries, outcome_observations
- tasks → context (optional parent), thread_entries (log), time_blocks (future), tags (many-to-many), outcome_observations
- raw_inputs → emails, transactions (future source types)
- clarification_items → any subject (task, context, email, raw_input)
- inactivity_checks → any subject (task, context)

## Tables

### contexts
```sql
CREATE TABLE contexts (
    context_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'closed')),
    summary       TEXT        NOT NULL DEFAULT '',
    last_event    TIMESTAMPTZ,
    last_thread_at TIMESTAMPTZ,
    debrief_status TEXT       NOT NULL DEFAULT 'pending' CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    outcome       TEXT        CHECK (outcome IN ('went_well', 'mixed', 'difficult', 'ongoing_issues')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (context_id)
);
```

### context_events
```sql
CREATE TABLE context_events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    kind          TEXT        NOT NULL,  -- note | email | transaction | task_created | task_completed | voice
    content       TEXT        NOT NULL,
    metadata      JSONB,
    source_id     UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_context_events_context ON context_events(context_id, created_at DESC);
```

### tasks
```sql
CREATE TABLE tasks (
    task_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'todo' CHECK (status IN ('todo', 'in_progress', 'done', 'cancelled')),
    priority      TEXT        NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    energy        TEXT        NOT NULL DEFAULT 'medium' CHECK (energy IN ('low', 'medium', 'high')),
    duration_min  INTEGER,
    due_date      TIMESTAMPTZ,
    scheduled_at  TIMESTAMPTZ,
    expected_update_days REAL,
    last_thread_at TIMESTAMPTZ,
    debrief_status TEXT       NOT NULL DEFAULT 'pending' CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    PRIMARY KEY (task_id)
);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_context ON tasks(context_id);
CREATE INDEX idx_tasks_due ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority);
```

### tags
```sql
CREATE TABLE tags (
    tag_id        UUID        NOT NULL DEFAULT gen_random_uuid(),
    name          TEXT        NOT NULL UNIQUE,
    PRIMARY KEY (tag_id)
);

CREATE TABLE task_tags (
    task_id       UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    tag_id        UUID        NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE context_tags (
    context_id    UUID        NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    tag_id        UUID        NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (context_id, tag_id)
);
```

### raw_inputs
```sql
CREATE TABLE raw_inputs (
    raw_input_id  UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type   TEXT        NOT NULL CHECK (source_type IN ('email', 'transaction', 'voice', 'file')),
    status        TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'processed', 'failed')),
    raw_content   TEXT        NOT NULL,
    processed_at  TIMESTAMPTZ,
    error         TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (raw_input_id)
);
CREATE INDEX idx_raw_inputs_status ON raw_inputs(status, created_at);
```

### emails
```sql
CREATE TABLE emails (
    email_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id  UUID        NOT NULL REFERENCES raw_inputs(raw_input_id),
    message_id    TEXT,
    from_address  TEXT        NOT NULL,
    from_name     TEXT,
    to_address    TEXT        NOT NULL,
    subject       TEXT        NOT NULL,
    body_text     TEXT        NOT NULL,
    body_html     TEXT,
    received_at   TIMESTAMPTZ NOT NULL,
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (email_id)
);
CREATE INDEX idx_emails_raw_input ON emails(raw_input_id);
CREATE INDEX idx_emails_context ON emails(context_id);
CREATE INDEX idx_emails_received ON emails(received_at DESC);
CREATE UNIQUE INDEX idx_emails_message_id ON emails(message_id) WHERE message_id IS NOT NULL;
```

### thread_entries
Unified thread model — polymorphic via `subject_type` + `subject_id`. Replaces the earlier per-entity `task_thread_entries` design.
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

### clarification_items
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

### inactivity_checks
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

### outcome_observations
Replaces the earlier `context_outcomes` design. Polymorphic — tracks observations for both tasks and contexts.
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

## Future Tables (not yet in migration)

### transactions
Phase 5 deliverable.
```sql
CREATE TABLE transactions (
    transaction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id   UUID        REFERENCES raw_inputs(raw_input_id),
    source         TEXT        NOT NULL,
    date           TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL,
    clean_name     TEXT,
    amount         INTEGER     NOT NULL,  -- cents, negative = debit
    category       TEXT,
    context_id     UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    notes          TEXT,
    reviewed       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (transaction_id)
);
```

### time_blocks
Phase 7 deliverable.
```sql
CREATE TABLE time_blocks (
    block_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    task_id     UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    calendar_id TEXT,
    confirmed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (block_id)
);
```
