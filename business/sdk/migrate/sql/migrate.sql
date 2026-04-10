-- Version: 1.01
-- Description: Create contexts table
CREATE TABLE contexts (
    context_id    UUID        NOT NULL DEFAULT gen_random_uuid(),
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    status        TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'closed')),
    summary       TEXT        NOT NULL DEFAULT '',
    last_event    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (context_id)
);

-- Version: 1.02
-- Description: Create context_events table
CREATE TABLE context_events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        NOT NULL REFERENCES contexts(context_id) ON DELETE CASCADE,
    kind          TEXT        NOT NULL,
    content       TEXT        NOT NULL,
    metadata      JSONB,
    source_id     UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_context_events_context ON context_events(context_id, created_at DESC);

-- Version: 1.03
-- Description: Create tasks table
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
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at  TIMESTAMPTZ,
    PRIMARY KEY (task_id)
);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_context ON tasks(context_id);
CREATE INDEX idx_tasks_due ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_priority ON tasks(priority);

-- Version: 1.04
-- Description: Create tags tables
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

-- Version: 1.05
-- Description: Create raw_inputs table
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

-- Version: 1.06
-- Description: Create emails table
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

-- Version: 1.07
-- Description: Create clarification_items table
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

-- Version: 1.08
-- Description: Create thread_entries table
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

-- Version: 1.09
-- Description: Create inactivity_checks table
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

-- Version: 1.10
-- Description: Create outcome_observations table
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

-- Version: 1.11
-- Description: Add thread/debrief columns to tasks and contexts
ALTER TABLE tasks ADD COLUMN expected_update_days REAL;
ALTER TABLE tasks ADD COLUMN last_thread_at TIMESTAMPTZ;
ALTER TABLE tasks ADD COLUMN debrief_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (debrief_status IN ('pending', 'done', 'skipped'));

ALTER TABLE contexts ADD COLUMN last_thread_at TIMESTAMPTZ;
ALTER TABLE contexts ADD COLUMN debrief_status TEXT NOT NULL DEFAULT 'pending'
    CHECK (debrief_status IN ('pending', 'done', 'skipped'));
ALTER TABLE contexts ADD COLUMN outcome TEXT
    CHECK (outcome IN ('went_well', 'mixed', 'difficult', 'ongoing_issues'));

-- Version: 1.12
-- Description: Create transactions table
CREATE TABLE transactions (
    transaction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    raw_input_id   UUID        REFERENCES raw_inputs(raw_input_id),
    source         TEXT        NOT NULL,
    date           TIMESTAMPTZ NOT NULL,
    description    TEXT        NOT NULL,
    clean_name     TEXT,
    amount         INTEGER     NOT NULL,
    category       TEXT,
    context_id     UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    notes          TEXT,
    reviewed       BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (transaction_id)
);
CREATE INDEX idx_transactions_date ON transactions(date DESC);
CREATE INDEX idx_transactions_context ON transactions(context_id);
CREATE INDEX idx_transactions_reviewed ON transactions(reviewed, created_at);
CREATE UNIQUE INDEX idx_transactions_dedup ON transactions(source, date, description, amount);

-- Version: 1.13
-- Description: Add task_debrief to clarification_items kind CHECK constraint
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    'context_assignment', 'stale_task', 'ambiguous_deadline',
    'new_context', 'overlapping_contexts', 'ambiguous_action',
    'voice_reference', 'inactivity_prompt', 'context_debrief',
    'task_debrief'
));

-- Version: 1.14
-- Description: Create events table
CREATE TABLE events (
    event_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    title         TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    location      TEXT,
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ NOT NULL,
    all_day       BOOLEAN     NOT NULL DEFAULT FALSE,
    raw_input_id  UUID        REFERENCES raw_inputs(raw_input_id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id)
);
CREATE INDEX idx_events_date ON events(starts_at, ends_at);
CREATE INDEX idx_events_context ON events(context_id);

-- Version: 1.15
-- Description: Create daily_plans and daily_plan_items tables
CREATE TABLE daily_plans (
    plan_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_date     DATE        NOT NULL,
    generation    INTEGER     NOT NULL DEFAULT 1,
    model_used    TEXT        NOT NULL,
    prompt_hash   TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plan_id)
);
CREATE INDEX idx_daily_plans_date ON daily_plans(plan_date DESC);

CREATE TABLE daily_plan_items (
    item_id             UUID        NOT NULL DEFAULT gen_random_uuid(),
    plan_id             UUID        NOT NULL REFERENCES daily_plans(plan_id) ON DELETE CASCADE,
    task_id             UUID        NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    position            INTEGER     NOT NULL,
    group_name          TEXT        NOT NULL DEFAULT 'ungrouped',
    group_position      INTEGER     NOT NULL DEFAULT 0,
    ai_duration_min     INTEGER,
    ai_priority_reason  TEXT,
    user_position       INTEGER,
    user_duration_min   INTEGER,
    status              TEXT        NOT NULL DEFAULT 'proposed' CHECK (status IN ('proposed', 'accepted', 'completed', 'dismissed')),
    dismiss_reason      TEXT        CHECK (dismiss_reason IN ('not_today', 'blocked', 'too_long', 'not_important', 'other')),
    dismiss_note        TEXT,
    completed_at        TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (item_id)
);
CREATE INDEX idx_daily_plan_items_plan ON daily_plan_items(plan_id, group_position, position);

-- Version: 1.16
-- Description: Create time_blocks table
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

-- Version: 1.17
-- Description: Task status simplification, task dependencies, context kind

-- 1. Add blocked_reason to tasks
ALTER TABLE tasks ADD COLUMN blocked_reason TEXT NOT NULL DEFAULT '';

-- 2. Add kind to contexts
ALTER TABLE contexts ADD COLUMN kind TEXT NOT NULL DEFAULT 'project'
    CHECK (kind IN ('project', 'area'));

-- 3. Create task_dependencies table
CREATE TABLE task_dependencies (
    task_id       UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    depends_on_id UUID NOT NULL REFERENCES tasks(task_id) ON DELETE CASCADE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (task_id, depends_on_id),
    CHECK (task_id != depends_on_id)
);
CREATE INDEX idx_task_deps_depends_on ON task_dependencies(depends_on_id);

-- 4. Drop old constraint before migrating data (old constraint rejects 'open'/'dismissed')
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;

-- 5. Migrate task statuses: todo→open, in_progress→open, cancelled→dismissed
UPDATE tasks SET status = 'open' WHERE status IN ('todo', 'in_progress');
UPDATE tasks SET status = 'dismissed' WHERE status = 'cancelled';

-- 6. Add new task status constraint
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN ('open', 'blocked', 'done', 'dismissed'));

-- Version: 1.18
-- Description: Phase 7c — notes, activity_logs, recurring tasks

-- 1. Notes table
CREATE TABLE notes (
    note_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    context_id    UUID        REFERENCES contexts(context_id) ON DELETE SET NULL,
    content       TEXT        NOT NULL,
    source        TEXT        NOT NULL DEFAULT 'manual' CHECK (source IN ('manual', 'voice', 'email')),
    raw_input_id  UUID        REFERENCES raw_inputs(raw_input_id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (note_id)
);
CREATE INDEX idx_notes_context ON notes(context_id);

-- 2. Note-tag junction (reuses existing tags table)
CREATE TABLE note_tags (
    note_id UUID NOT NULL REFERENCES notes(note_id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(tag_id) ON DELETE CASCADE,
    PRIMARY KEY (note_id, tag_id)
);

-- 3. Activity logs (generic tracking for streaks/habits)
CREATE TABLE activity_logs (
    log_id       UUID        NOT NULL DEFAULT gen_random_uuid(),
    subject_type TEXT        NOT NULL CHECK (subject_type IN ('task', 'note')),
    subject_id   UUID        NOT NULL,
    value        TEXT,
    logged_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (log_id)
);
CREATE INDEX idx_activity_logs_subject ON activity_logs(subject_type, subject_id);
CREATE INDEX idx_activity_logs_logged  ON activity_logs(logged_at);

-- 4. Recurring tasks (extend existing tasks table)
ALTER TABLE tasks ADD COLUMN recurrence_rule TEXT;
ALTER TABLE tasks ADD COLUMN recurrence_parent_id UUID REFERENCES tasks(task_id);

-- Version: 1.19
-- Description: Add retry scheduling fields to raw_inputs for async ingest pipeline
ALTER TABLE raw_inputs
    ADD COLUMN retry_count   INT         NOT NULL DEFAULT 0,
    ADD COLUMN next_retry_at TIMESTAMPTZ,
    ADD COLUMN max_retries   INT         NOT NULL DEFAULT 5;

CREATE INDEX idx_raw_inputs_retryable ON raw_inputs(created_at)
    WHERE status = 'pending';

-- Version: 1.20
-- Description: Create entity_links table for cross-entity semantic linking
CREATE TABLE entity_links (
    link_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type  TEXT        NOT NULL CHECK (source_type IN ('task', 'note', 'event')),
    source_id    UUID        NOT NULL,
    target_type  TEXT        NOT NULL CHECK (target_type IN ('task', 'note', 'event')),
    target_id    UUID        NOT NULL,
    confidence   FLOAT8      NOT NULL DEFAULT 1.0,
    kind         TEXT        NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'ai_suggested')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (link_id),
    CONSTRAINT entity_links_no_self_link CHECK (
        NOT (source_type = target_type AND source_id = target_id)
    )
);
CREATE UNIQUE INDEX idx_entity_links_pair   ON entity_links(source_type, source_id, target_type, target_id);
CREATE INDEX        idx_entity_links_source ON entity_links(source_type, source_id);
CREATE INDEX        idx_entity_links_target ON entity_links(target_type, target_id);

-- Version: 1.21
-- Description: Add task_id to notes table for task-level note attachment
ALTER TABLE notes ADD COLUMN task_id UUID REFERENCES tasks(task_id) ON DELETE SET NULL;

-- Add CHECK constraint that at least one of context_id or task_id is non-null
ALTER TABLE notes ADD CONSTRAINT notes_has_target CHECK (context_id IS NOT NULL OR task_id IS NOT NULL);

-- Add index on task_id
CREATE INDEX idx_notes_task ON notes(task_id);

-- Version: 1.22
-- Description: Add pipeline result tracking to raw_inputs
ALTER TABLE raw_inputs ADD COLUMN result JSONB;


-- Version: 1.23
-- Description: Add parent_context_id for area hierarchy
ALTER TABLE contexts ADD COLUMN parent_context_id UUID REFERENCES contexts(context_id) ON DELETE SET NULL;
CREATE INDEX idx_contexts_parent ON contexts(parent_context_id);

-- Version: 1.24
-- Description: Add track_outcome flag to tasks
ALTER TABLE tasks ADD COLUMN track_outcome BOOLEAN NOT NULL DEFAULT false;

-- Version: 1.25
-- Description: Drop legacy context_events table
DROP INDEX IF EXISTS idx_context_events_context;
DROP TABLE IF EXISTS context_events;

-- Version: 1.26
-- Description: Fix stale clarification kind CHECK, add weekly_review kind and week subject_type
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    'context_assignment', 'stale_task', 'ambiguous_deadline',
    'new_context', 'overlapping_contexts', 'ambiguous_action',
    'voice_reference', 'inactivity_prompt', 'context_debrief',
    'task_debrief', 'entity_link', 'weekly_review'
));

ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_subject_type_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_subject_type_check CHECK (subject_type IN (
    'task', 'context', 'email', 'raw_input', 'week'
));

-- Version: 1.27
-- Description: Add index for habit grid bulk queries
CREATE INDEX IF NOT EXISTS idx_activity_logs_subject_date ON activity_logs(subject_type, subject_id, logged_at);

-- Version: 1.28
-- Description: Add unconfirmed columns + classification_corrections table + type_assignment clarification kind

-- 1. Add unconfirmed column to tasks
ALTER TABLE tasks ADD COLUMN unconfirmed BOOLEAN NOT NULL DEFAULT false;

-- 2. Add unconfirmed column to notes
ALTER TABLE notes ADD COLUMN unconfirmed BOOLEAN NOT NULL DEFAULT false;

-- 3. Add unconfirmed column to events
ALTER TABLE events ADD COLUMN unconfirmed BOOLEAN NOT NULL DEFAULT false;

-- 4. Create classification_corrections table
CREATE TABLE classification_corrections (
    correction_id UUID        NOT NULL DEFAULT gen_random_uuid(),
    clause_text   TEXT        NOT NULL,
    predicted_type TEXT       NOT NULL,
    confidence    FLOAT8      NOT NULL,
    actual_type   TEXT        NOT NULL,
    source        TEXT        NOT NULL CHECK (source IN ('clarification_answered', 'correction_applied')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (correction_id)
);
CREATE INDEX idx_classification_corrections_source ON classification_corrections(source, created_at DESC);
CREATE INDEX idx_classification_corrections_actual_type ON classification_corrections(actual_type);

-- 5. Add type_assignment to clarification_items kind CHECK constraint
ALTER TABLE clarification_items DROP CONSTRAINT IF EXISTS clarification_items_kind_check;
ALTER TABLE clarification_items ADD CONSTRAINT clarification_items_kind_check CHECK (kind IN (
    'context_assignment', 'stale_task', 'ambiguous_deadline',
    'new_context', 'overlapping_contexts', 'ambiguous_action',
    'voice_reference', 'inactivity_prompt', 'context_debrief',
    'task_debrief', 'entity_link', 'weekly_review', 'type_assignment'
));
