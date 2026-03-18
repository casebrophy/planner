# Data Model

Three top-level concepts: **contexts** (ongoing situations), **tasks** (discrete actions), **sources** (external data).

## Entity Relationships

- contexts → tasks (one-to-many, optional), context_events (timeline), raw_inputs, tags (many-to-many)
- tasks → context (optional parent), thread_entries (log), time_blocks, tags (many-to-many)
- raw_inputs → emails, transactions (future source types)

## Tables

### contexts
```sql
CREATE TABLE contexts (
    id           TEXT PRIMARY KEY,          -- UUID
    title        TEXT NOT NULL,             -- "Home renovation", "2024 taxes"
    description  TEXT NOT NULL DEFAULT '',  -- What this context is about
    status       TEXT NOT NULL DEFAULT 'active',  -- active | paused | closed
    summary      TEXT NOT NULL DEFAULT '',  -- Claude-maintained rolling summary
    last_event   TEXT,                      -- Timestamp of last activity
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
);
```

### context_events
```sql
CREATE TABLE context_events (
    id           TEXT PRIMARY KEY,
    context_id   TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL,  -- note | email | transaction | task_created | task_completed | voice
    content      TEXT NOT NULL,  -- Human-readable description of what happened
    metadata     TEXT,           -- JSON blob: source-specific fields (email ID, amount, etc.)
    source_id    TEXT,           -- Optional link to raw_inputs.id
    created_at   TEXT NOT NULL
);
```

### tasks
```sql
CREATE TABLE tasks (
    id           TEXT PRIMARY KEY,
    context_id   TEXT REFERENCES contexts(id) ON DELETE SET NULL,  -- nullable: standalone tasks
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'todo',     -- todo | in_progress | done | cancelled
    priority     TEXT NOT NULL DEFAULT 'medium',   -- low | medium | high | urgent
    energy       TEXT NOT NULL DEFAULT 'medium',   -- low | medium | high (mental effort required)
    duration_min INTEGER,                          -- Estimated minutes, null if unknown
    due_date     TEXT,                             -- ISO date, nullable
    scheduled_at TEXT,                             -- ISO datetime when actually scheduled
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL,
    completed_at TEXT
);
```

### thread_entries
```sql
CREATE TABLE thread_entries (
    id              TEXT PRIMARY KEY,
    subject_type    TEXT NOT NULL,   -- task | context
    subject_id      TEXT NOT NULL,
    kind            TEXT NOT NULL,   -- update | blocker | blocker_resolved | milestone
                                     -- | scope_change | timeline_slip | external_dep
                                     -- | decision | observation | email | transaction
    content         TEXT NOT NULL,
    metadata        TEXT,            -- JSON: extracted fields (blocking_party, etc.)
    source          TEXT NOT NULL,   -- user | voice | email | transaction | system | claude
    source_id       TEXT,
    sentiment       TEXT,
    requires_action INTEGER DEFAULT 0,
    created_at      TEXT NOT NULL
);
```

### time_blocks
```sql
CREATE TABLE time_blocks (
    id          TEXT PRIMARY KEY,
    task_id     TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    starts_at   TEXT NOT NULL,   -- ISO datetime
    ends_at     TEXT NOT NULL,   -- ISO datetime
    calendar_id TEXT,            -- External calendar event ID if synced
    confirmed   INTEGER NOT NULL DEFAULT 0,  -- 0 = proposed, 1 = confirmed by user
    created_at  TEXT NOT NULL
);
```

### raw_inputs
```sql
CREATE TABLE raw_inputs (
    id           TEXT PRIMARY KEY,
    source_type  TEXT NOT NULL,   -- email | transaction | voice | file
    status       TEXT NOT NULL DEFAULT 'pending',  -- pending | processing | processed | failed
    raw_content  TEXT NOT NULL,   -- Full raw content (email body, transaction JSON, etc.)
    processed_at TEXT,
    error        TEXT,            -- If processing failed, why
    created_at   TEXT NOT NULL
);
```

### emails
```sql
CREATE TABLE emails (
    id           TEXT PRIMARY KEY,
    raw_input_id TEXT NOT NULL REFERENCES raw_inputs(id),
    message_id   TEXT,            -- Email Message-ID header
    from_address TEXT NOT NULL,
    from_name    TEXT,
    subject      TEXT NOT NULL,
    body_text    TEXT NOT NULL,   -- Plain text body
    body_html    TEXT,            -- HTML body (stored but rarely used)
    received_at  TEXT NOT NULL,
    context_id   TEXT REFERENCES contexts(id) ON DELETE SET NULL  -- Assigned after processing
);
```

### transactions
```sql
CREATE TABLE transactions (
    id             TEXT PRIMARY KEY,
    raw_input_id   TEXT REFERENCES raw_inputs(id),
    source         TEXT NOT NULL,          -- "chase_checking", "amex_gold", etc.
    date           TEXT NOT NULL,          -- ISO date
    description    TEXT NOT NULL,          -- Original bank description
    clean_name     TEXT,                   -- Claude-cleaned merchant name
    amount         INTEGER NOT NULL,       -- Cents, negative = debit
    category       TEXT,                   -- Claude-assigned category
    context_id     TEXT REFERENCES contexts(id) ON DELETE SET NULL,
    notes          TEXT,
    reviewed       INTEGER NOT NULL DEFAULT 0,  -- Has user confirmed the categorisation
    created_at     TEXT NOT NULL
);
```

### tags
```sql
CREATE TABLE tags (
    id   TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE task_tags (
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    tag_id  TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (task_id, tag_id)
);

CREATE TABLE context_tags (
    context_id TEXT NOT NULL REFERENCES contexts(id) ON DELETE CASCADE,
    tag_id     TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (context_id, tag_id)
);
```

## Indexes
```sql
-- Frequent list queries
CREATE INDEX idx_tasks_status    ON tasks(status);
CREATE INDEX idx_tasks_context   ON tasks(context_id);
CREATE INDEX idx_tasks_due       ON tasks(due_date) WHERE due_date IS NOT NULL;
CREATE INDEX idx_tasks_scheduled ON tasks(scheduled_at) WHERE scheduled_at IS NOT NULL;

-- Context lookups
CREATE INDEX idx_events_context  ON context_events(context_id, created_at DESC);
CREATE INDEX idx_emails_context  ON emails(context_id);
CREATE INDEX idx_txns_context    ON transactions(context_id);
CREATE INDEX idx_txns_date       ON transactions(date DESC);

-- Pipeline processing
CREATE INDEX idx_inputs_status   ON raw_inputs(status, created_at);

-- Full-text search (SQLite FTS5)
CREATE VIRTUAL TABLE tasks_fts USING fts5(
    title, description, content=tasks, content_rowid=rowid
);
CREATE VIRTUAL TABLE contexts_fts USING fts5(
    title, description, summary, content=contexts, content_rowid=rowid
);
```

## Thread & Feedback Additions
```sql
-- Added to tasks table
ALTER TABLE tasks ADD COLUMN expected_update_hours INTEGER NOT NULL DEFAULT 96;
ALTER TABLE tasks ADD COLUMN last_activity_at TEXT;

-- Thread entries (replaces plain task_notes for in-progress tasks)
CREATE TABLE task_thread_entries (
    id         TEXT PRIMARY KEY,
    task_id    TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    content    TEXT NOT NULL,
    kind       TEXT NOT NULL DEFAULT 'update',
    source     TEXT NOT NULL DEFAULT 'user',
    structured TEXT,
    created_at TEXT NOT NULL
);

-- Debrief outcomes on closed contexts
CREATE TABLE context_outcomes (
    id              TEXT PRIMARY KEY,
    context_id      TEXT NOT NULL REFERENCES contexts(id),
    outcome         TEXT NOT NULL,
    primary_reason  TEXT,
    blockers        TEXT,
    avoidable       INTEGER,
    duration_delta  REAL,
    cost_delta      REAL,
    lessons         TEXT,
    debrief_skipped INTEGER NOT NULL DEFAULT 0,
    created_at      TEXT NOT NULL
);
```
