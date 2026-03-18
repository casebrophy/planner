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
