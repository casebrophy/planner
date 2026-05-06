# Data Model

Five top-level concepts: **contexts** (ongoing situations), **tasks** (discrete actions), **events** (fixed commitments), **notes** (knowledge capture), **sources** (external data).

Database: **PostgreSQL** with the `vector` extension (pgvector) for embeddings. Mapped to port 5433 locally via Docker.

Schema source of truth: `business/sdk/migrate/sql/migrate.sql` (currently v1.47). This document captures the live shape; consult the migration file for the full version history.

## Entity Relationships

- contexts → tasks (one-to-many, optional), thread_entries, outcome_observations, emails, tags (many-to-many), notes, events, transactions, child contexts (self-referential via `parent_context_id`)
- tasks → context (optional parent), thread_entries (log), time_blocks, tags (many-to-many), outcome_observations, daily_plan_items, activity_logs, notes (per-task), entity_links, recurrence_parent (self-referential), raw_input (optional source)
- events → context (optional), entity_links, raw_input (optional source) — fixed time commitments that constrain daily plan
- daily_plans → daily_plan_items (one plan per day per generation)
- notes → context (optional), task (optional), tags (many-to-many via note_tags), entity_links, activity_logs, raw_input (optional source)
- raw_inputs → emails, transactions, tasks, notes, events (entity products); supports retry, reingest, and reclassification
- clarification_items → any subject (task, context, email, raw_input, week, note, event)
- inactivity_checks → any subject (task, context)
- entity_links → any (task, note, event) ↔ any (task, note, event)
- embeddings → any subject (polymorphic)
- transaction_splits → transaction
- classification_corrections → standalone training data (no FKs)

## Tables

### contexts
```sql
CREATE TABLE contexts (
    context_id        UUID        NOT NULL DEFAULT gen_random_uuid(),
    title             TEXT        NOT NULL,
    description       TEXT        NOT NULL DEFAULT '',
    kind              TEXT        NOT NULL DEFAULT 'project' CHECK (kind IN ('project', 'area', 'list')),
    status            TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'closed')),
    summary           TEXT        NOT NULL DEFAULT '',
    last_event        TIMESTAMPTZ,
    last_thread_at    TIMESTAMPTZ,
    debrief_status    TEXT        NOT NULL DEFAULT 'pending' CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    outcome           TEXT        CHECK (outcome IN ('went_well', 'mixed', 'difficult', 'ongoing_issues')),
    parent_context_id UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (context_id)
);
CREATE INDEX idx_contexts_parent ON contexts(parent_context_id);
```

> The legacy `context_events` table was retired in v1.25. Context timelines are now derived from `thread_entries` filtered on `subject_type='context'`.

### tasks
```sql
CREATE TABLE tasks (
    task_id              UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id           UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title                TEXT        NOT NULL,
    description          TEXT        NOT NULL DEFAULT '',
    status               TEXT        NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'blocked', 'done', 'dismissed')),
    priority             TEXT        NOT NULL DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'urgent')),
    energy               TEXT        NOT NULL DEFAULT 'medium' CHECK (energy IN ('low', 'medium', 'high')),
    duration_min         INTEGER,
    due_date             TIMESTAMPTZ,
    scheduled_at         TIMESTAMPTZ,
    expected_update_days REAL,
    last_thread_at       TIMESTAMPTZ,
    debrief_status       TEXT        NOT NULL DEFAULT 'pending' CHECK (debrief_status IN ('pending', 'done', 'skipped')),
    blocked_reason       TEXT        NOT NULL DEFAULT '',
    recurrence_rule      TEXT,
    recurrence_parent_id UUID        REFERENCES tasks(task_id) ON DELETE SET NULL,
    track_outcome        BOOLEAN     NOT NULL DEFAULT FALSE,
    unconfirmed          BOOLEAN     NOT NULL DEFAULT FALSE,
    raw_input_id         UUID        REFERENCES raw_inputs(raw_input_id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at         TIMESTAMPTZ,
    PRIMARY KEY (task_id)
);
CREATE INDEX idx_tasks_status    ON tasks(status);
CREATE INDEX idx_tasks_context   ON tasks(context_id);
CREATE INDEX idx_tasks_due       ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority  ON tasks(priority);
CREATE INDEX idx_tasks_raw_input ON tasks(raw_input_id);
```

### task_dependencies
```sql
CREATE TABLE task_dependencies (
    task_id       UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on_id);
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
    raw_input_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type        TEXT        NOT NULL CHECK (source_type IN ('email', 'transaction', 'voice', 'file', 'manual')),
    status             TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'processed', 'partial', 'failed')),
    raw_content        TEXT        NOT NULL,
    processed_at       TIMESTAMPTZ,
    error              TEXT,
    retry_count        INT         NOT NULL DEFAULT 0,
    next_retry_at      TIMESTAMPTZ,
    max_retries        INT         NOT NULL DEFAULT 5,
    result             JSONB,                          -- pipeline result envelope (per-stage outcomes)
    user_correction    TEXT,                           -- freeform correction text from reclassification
    source_entity_id   UUID,                           -- if reingest, the originating entity
    source_entity_kind TEXT,                           -- 'task' | 'note' | 'event'
    skip_classify      BOOLEAN     NOT NULL DEFAULT FALSE,
    reingest_mode      BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (raw_input_id)
);
CREATE INDEX idx_raw_inputs_status        ON raw_inputs(status, created_at);
CREATE INDEX idx_raw_inputs_retryable     ON raw_inputs(created_at) WHERE status = 'pending';
CREATE INDEX idx_raw_inputs_source_entity ON raw_inputs(source_entity_kind, source_entity_id) WHERE source_entity_id IS NOT NULL;
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
CREATE INDEX idx_emails_context   ON emails(context_id);
CREATE INDEX idx_emails_received  ON emails(received_at DESC);
CREATE UNIQUE INDEX idx_emails_message_id ON emails(message_id) WHERE message_id IS NOT NULL;
```

### thread_entries
Unified thread model — polymorphic via `subject_type` + `subject_id`. Replaces both the earlier per-entity `task_thread_entries` design and the legacy `context_events` table.
```sql
CREATE TABLE thread_entries (
    entry_id         UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type     TEXT        NOT NULL CHECK (subject_type IN ('task', 'context', 'raw_input')),
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
CREATE INDEX idx_thread_action  ON thread_entries(requires_action) WHERE requires_action = TRUE;
```

### clarification_items
```sql
CREATE TABLE clarification_items (
    clarification_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    kind                TEXT        NOT NULL CHECK (kind IN (
        'context_assignment', 'stale_task', 'ambiguous_deadline',
        'new_context', 'overlapping_contexts', 'ambiguous_action',
        'voice_reference', 'inactivity_prompt', 'context_debrief',
        'task_debrief', 'entity_link', 'weekly_review', 'type_assignment',
        'event_prep', 'ambiguous_entity_match', 'knowledge_gap'
    )),
    status              TEXT        NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'snoozed', 'resolved', 'dismissed')),
    subject_type        TEXT        NOT NULL CHECK (subject_type IN ('task', 'context', 'email', 'raw_input', 'week', 'note', 'event')),
    subject_id          UUID        NOT NULL,
    subject_description TEXT        NOT NULL DEFAULT '',
    question            TEXT        NOT NULL,
    claude_guess        JSONB,
    reasoning           TEXT,
    answer_options      JSONB       NOT NULL,
    answer              JSONB,
    priority_score      REAL        NOT NULL DEFAULT 0.0,
    snoozed_until       TIMESTAMPTZ,
    suppress_until      TIMESTAMPTZ,                              -- dismissed knowledge_gap suppression window
    gap_category        TEXT        NOT NULL DEFAULT '',          -- per-category dedup for knowledge_gap
    source_hash         TEXT        DEFAULT '',                   -- ingestion-derived dedup key (kind, source_hash)
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at         TIMESTAMPTZ,
    PRIMARY KEY (clarification_id),
    CONSTRAINT uq_clarification_dedup UNIQUE (kind, subject_type, subject_id, gap_category)
);
CREATE INDEX        idx_clarification_pending      ON clarification_items(status, priority_score DESC) WHERE status = 'pending';
CREATE INDEX        idx_clarification_snoozed      ON clarification_items(snoozed_until) WHERE status = 'snoozed';
CREATE INDEX        idx_clarification_subject      ON clarification_items(subject_type, subject_id);
CREATE UNIQUE INDEX idx_clarification_source_dedup ON clarification_items(kind, source_hash) WHERE source_hash != '';
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
        'lesson', 'completion_pattern', 'scope_change', 'cost_profile',
        'debrief'
    )),
    data             JSONB       NOT NULL,
    source           TEXT        NOT NULL CHECK (source IN ('user', 'inferred')),
    confidence       REAL        NOT NULL DEFAULT 1.0,
    weight           REAL        NOT NULL DEFAULT 1.0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (observation_id)
);
CREATE INDEX idx_observations_subject ON outcome_observations(subject_type, subject_id);
CREATE INDEX idx_observations_kind    ON outcome_observations(kind, created_at DESC);
```

### transactions
```sql
CREATE TABLE transactions (
    transaction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id   UUID        REFERENCES raw_inputs(raw_input_id),
    source         TEXT        NOT NULL,
    date           TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL,
    clean_name     TEXT,
    amount         INTEGER     NOT NULL,                          -- cents, negative = debit
    category       TEXT,
    context_id     UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    notes          TEXT,
    reviewed       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (transaction_id)
);

CREATE INDEX idx_transactions_date     ON transactions(date DESC);
CREATE INDEX idx_transactions_context  ON transactions(context_id);
CREATE INDEX idx_transactions_reviewed ON transactions(reviewed, created_at);
CREATE UNIQUE INDEX idx_transactions_dedup ON transactions(source, date, description, amount);
```

### transaction_splits
Splits one transaction across multiple parties (e.g. roommate venmo).
```sql
CREATE TABLE transaction_splits (
    split_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    transaction_id UUID        NOT NULL REFERENCES transactions(transaction_id) ON DELETE CASCADE,
    party_name     TEXT        NOT NULL,
    amount         INTEGER     NOT NULL,                          -- cents
    venmo_handle   TEXT,
    settled        BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (split_id)
);
CREATE INDEX idx_transaction_splits_transaction_id ON transaction_splits(transaction_id);
```

### events
Fixed commitments (appointments, trips, meetings) — not tasks. Constrain daily plan generation.
```sql
CREATE TABLE events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    location      TEXT,                                           -- freeform; Claude reasons about travel/portability
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    all_day       BOOLEAN     NOT NULL DEFAULT FALSE,
    raw_input_id  UUID        REFERENCES raw_inputs(raw_input_id),
    unconfirmed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_events_date    ON events(starts_at, ends_at);
CREATE INDEX idx_events_context ON events(context_id);
```

### daily_plans
One row per generated plan.
```sql
CREATE TABLE daily_plans (
    plan_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_date     DATE        NOT NULL,
    generation    INTEGER     NOT NULL DEFAULT 1,                 -- increments on regenerate
    model_used    TEXT        NOT NULL,                            -- which Claude model generated this
    prompt_hash   TEXT,                                            -- hash of prompt for reproducibility
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plan_id)
);
CREATE INDEX idx_daily_plans_date ON daily_plans(plan_date DESC);
```

### daily_plan_items
Each task in a daily plan, with AI estimates and user overrides.
```sql
CREATE TABLE daily_plan_items (
    item_id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_id             UUID        NOT NULL REFERENCES daily_plans(plan_id) ON DELETE CASCADE,
    task_id             UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    position            INTEGER     NOT NULL,                          -- ordering within group
    group_name          TEXT        NOT NULL DEFAULT 'ungrouped',      -- AI-assigned group (e.g. "errands", "deep work", context title)
    group_position      INTEGER     NOT NULL DEFAULT 0,                -- ordering of groups
    ai_duration_min     INTEGER,                                       -- AI-estimated duration
    ai_priority_reason  TEXT,                                          -- why AI chose this priority/position
    user_position       INTEGER,                                       -- set when user drags to reorder
    user_duration_min   INTEGER,                                       -- set when user overrides estimate
    status              TEXT        NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'accepted', 'completed', 'dismissed')),
    dismiss_reason      TEXT        CHECK (dismiss_reason IN ('not_today', 'blocked', 'too_long', 'not_important', 'other')),
    dismiss_note        TEXT,                                          -- freeform reason
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id)
);
CREATE INDEX idx_daily_plan_items_plan ON daily_plan_items(plan_id, group_position, position);
```

### time_blocks
Time-slotted task scheduling.
```sql
CREATE TABLE time_blocks (
    block_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    task_id     UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    confirmed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (block_id)
);
```

### notes
Freestanding knowledge capture — facts, ideas, preferences. Tags provide emergent topic grouping; auto-tagged by ingestion pipeline, user-editable. A note may attach to a context, a task, or stand alone (the historical `notes_has_target` constraint was dropped in v1.44 to support task→note reclassification).
```sql
CREATE TABLE notes (
    note_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    task_id       UUID        REFERENCES tasks(task_id) ON DELETE SET NULL,
    content       TEXT        NOT NULL,
    source        TEXT        NOT NULL DEFAULT 'manual' CHECK (source IN (
        'manual', 'voice', 'email', 'clarification', 'reclassified_from_task', 'correction'
    )),
    raw_input_id  UUID        REFERENCES raw_inputs(raw_input_id),
    unconfirmed   BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (note_id)
);
CREATE INDEX idx_notes_context ON notes(context_id);
CREATE INDEX idx_notes_task    ON notes(task_id);

CREATE TABLE note_tags (
    note_id       UUID        NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
    tag_id        UUID        NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);
```

### activity_logs
Phase 7c deliverable. Generic tracking log for habits, streaks, and history. Any entity (note, task, etc.) can accumulate log entries over time.
```sql
CREATE TABLE activity_logs (
    log_id        UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type  TEXT        NOT NULL CHECK (subject_type IN ('task', 'note')),
    subject_id    UUID        NOT NULL,
    value         TEXT,
    logged_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (log_id)
);
CREATE INDEX idx_activity_logs_subject       ON activity_logs(subject_type, subject_id);
CREATE INDEX idx_activity_logs_logged        ON activity_logs(logged_at);
CREATE INDEX idx_activity_logs_subject_date  ON activity_logs(subject_type, subject_id, logged_at);
```

### entity_links
First-class typed relationships between any two entities (Phase 7d). Bidirectional in concept; queried both as source and target.
```sql
CREATE TABLE entity_links (
    link_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type  TEXT        NOT NULL CHECK (source_type IN ('task', 'note', 'event')),
    source_id    UUID        NOT NULL,
    target_type  TEXT        NOT NULL CHECK (target_type IN ('task', 'note', 'event')),
    target_id    UUID        NOT NULL,
    confidence   FLOAT8      NOT NULL DEFAULT 1.0,
    kind         TEXT        NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'ai_suggested', 'knowledge_gap_answer')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (link_id),
    CONSTRAINT entity_links_no_self_link CHECK (NOT (source_type = target_type AND source_id = target_id))
);
CREATE UNIQUE INDEX idx_entity_links_pair   ON entity_links(source_type, source_id, target_type, target_id);
CREATE INDEX        idx_entity_links_source ON entity_links(source_type, source_id);
CREATE INDEX        idx_entity_links_target ON entity_links(target_type, target_id);
```

### embeddings
pgvector storage for semantic search (Phase 6). 1024-dim vectors for the `qwen3-embedding` model. Index resized in v1.39 (768→1024 dims) and switched between HNSW (v1.32) and ivfflat (v1.39 current) as model dimensions changed.
```sql
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE embeddings (
    embedding_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type   TEXT         NOT NULL,
    source_id     UUID         NOT NULL,
    content       TEXT         NOT NULL,
    embedding     vector(1024) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_embeddings_source ON embeddings(source_type, source_id);
CREATE INDEX idx_embeddings_vector ON embeddings USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
```

### classification_corrections
Training data for the classifier — captures (predicted_type, actual_type) pairs whenever a clarification is answered or a user-applied correction reclassifies an entity. Standalone (no FKs); feeds the Phase 7c+ feedback loop.
```sql
CREATE TABLE classification_corrections (
    correction_id  UUID        NOT NULL DEFAULT gen_random_uuid(),
    clause_text    TEXT        NOT NULL,
    predicted_type TEXT        NOT NULL,
    confidence     FLOAT8      NOT NULL,
    actual_type    TEXT        NOT NULL,
    source         TEXT        NOT NULL CHECK (source IN ('clarification_answered', 'correction_applied')),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (correction_id)
);
CREATE INDEX idx_classification_corrections_source       ON classification_corrections(source, created_at DESC);
CREATE INDEX idx_classification_corrections_actual_type  ON classification_corrections(actual_type);
```
