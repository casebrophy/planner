# Clarification queue & pattern recognition

The **clarification queue** surfaces gaps Claude can't resolve automatically as swipeable cards worked through on your schedule. The **pattern recognition** layer builds behavioral and situational insights once the queue has been running long enough.

---

## Clarification Queue

### Card generation triggers

| Source | Trigger condition |
|--------|------------------|
| Pipeline | Transaction matched to context with confidence < 70% |
| Pipeline | Email action item ambiguous (pleasantry vs. task) |
| Pipeline | Email linked to context tentatively, sender domain unknown |
| Pipeline | Extracted deadline ambiguous ("end of month" without clear month) |
| Pipeline | Receipt items could belong to multiple contexts |
| Pipeline | New context auto-created — confirm Claude's chosen title/description |
| Context engine | Task in `todo` for N+ days with no activity |
| Context engine | Context has had no events in 30+ days |
| Context engine | Two contexts appear to overlap — merge candidate |
| Context engine | Task estimated duration inconsistent with description |
| MCP interaction | Voice capture with ambiguous intent |
| MCP interaction | Follow-up message with uncertain reference resolution |

### Card anatomy (fields)

- **source_type + age** — what produced the card and how long it has been waiting; ages visibly
- **fact** — the specific uncertain thing, stated plainly; never a wall of text
- **claude_guess** — prominently shown; most cards the guess is correct and answering is one tap
- **reasoning** — collapsed by default; Claude's explanation of its guess; expand on tap
- **answer_options** — context-sensitive per card type (see table below)
- **snooze** — defers 24 hours; not dismissal; card returns

### Answer options by card type

| Card type | Option 1 | Option 2 | Option 3 |
|-----------|----------|----------|----------|
| Context assignment | ✓ Correct | ✗ Wrong → pick | Create new context |
| Stale task | Still relevant | Cancel it | Snooze 1 week |
| Ambiguous deadline | Date chips (Mon/Fri/EOM) | No deadline | Enter date |
| New auto-context | ✓ Looks right | Edit title/desc | Merge with existing |
| Overlapping contexts | Keep separate | Merge → pick primary | Dismiss |
| Ambiguous action item | ✓ Is a task | ✗ Not a task | Edit before saving |
| Voice reference uncertain | ✓ Right receipt | ✗ Wrong → pick | Not about anything |
| Inactivity prompt | Still in progress + update | It's done | Blocked / Deprioritised |
| Context debrief | How did it go? (scale) | See debrief flow | Skip debrief |

### Session / ordering rules

- Queue is persistent in the database; enter and exit at any time; progress saved per card
- Cards ordered by: `priority_score = age_hours × 0.4 + amount_relevance × 0.3 + context_completeness × 0.3`
- `context_completeness`: cards that complete a gap in an active context rank higher
- Count shown as a badge on Today tab (mobile) and dashboard (web) — no push notifications

### "By the way" section

- Shown after clearing the queue or answering 5+ cards in a session
- Brief summary of what was resolved and any notable patterns across the batch
- Natural entry point for pattern surfacing — a few lines, no separate screen, no action required

---

## Pattern Recognition

### Three-layer summary

| Layer | When available | What it does | What it feeds |
|-------|---------------|-------------|---------------|
| 1 — Statistical summaries | Day one | SQL aggregations: completion rates, duration accuracy, overdue rates, creation sources, context lifetimes, time-of-day patterns | "By the way" section; task duration estimates |
| 2 — Clustering / archetypes | ~4 weeks | Embedding clusters reveal recurring task/context archetypes (e.g. "short admin tasks that always get deprioritised") | Duration estimation for new tasks; clarification queue ordering |
| 3 — Situational matching | 2–3 months of closed contexts | Embedding search over closed contexts for structurally similar historical situations; generates synthesis of what happened and what's worth knowing | Context detail "Similar situations" section |

### Surfacing locations

- **End of clarification session** — "by the way" section; Layer 1 insights + situational matches for contexts reviewed
- **Context detail view** — "similar situations" section at the bottom; Layer 3; collapsed by default
- **Task creation / editing** — Layer 1 and 2 inform duration estimate; indicator if task type has poor completion history

---

## Data Model

### clarification_items

```sql
CREATE TABLE clarification_items (
    id              TEXT PRIMARY KEY,
    kind            TEXT NOT NULL,     -- transaction_assignment | stale_task | ambiguous_deadline
                                       -- | new_context | overlapping_contexts | ambiguous_action
                                       -- | voice_reference | context_status
    status          TEXT NOT NULL DEFAULT 'pending',  -- pending | snoozed | resolved | dismissed
    subject_type    TEXT NOT NULL,     -- task | context | transaction | email | raw_input
    subject_id      TEXT NOT NULL,     -- FK to the subject record
    question        TEXT NOT NULL,     -- the question as shown on the card
    claude_guess    TEXT,              -- JSON: claude's best guess and confidence
    reasoning       TEXT,              -- claude's explanation (shown collapsed)
    answer_options  TEXT NOT NULL,     -- JSON array of {label, action, payload}
    answer          TEXT,              -- JSON: the answer given (null until resolved)
    priority_score  REAL NOT NULL DEFAULT 0.0,
    snoozed_until   TEXT,
    created_at      TEXT NOT NULL,
    resolved_at     TEXT
);

CREATE INDEX idx_clarification_pending  ON clarification_items(status, priority_score DESC)
    WHERE status = 'pending';
CREATE INDEX idx_clarification_snoozed  ON clarification_items(snoozed_until)
    WHERE status = 'snoozed';
```

### pattern_observations

```sql
CREATE TABLE pattern_observations (
    id           TEXT PRIMARY KEY,
    kind         TEXT NOT NULL,     -- stat_summary | cluster_insight | situational_match
    subject_type TEXT,              -- what this pattern is about (context, task_type, etc.)
    subject_id   TEXT,              -- specific subject if applicable
    content      TEXT NOT NULL,     -- JSON: the computed pattern data
    generated_by TEXT NOT NULL,     -- 'sql' | 'claude' | 'ml_service'
    valid_until  TEXT NOT NULL,     -- when to recompute
    created_at   TEXT NOT NULL
);
```

---

## MCP Tools

| Tool | Description |
|------|-------------|
| `get_clarification_queue` | Returns pending clarification items ordered by priority |
| `resolve_clarification` | Submits an answer to a clarification item; applies the resolution and marks it resolved |
| `snooze_clarification` | Snoozes an item for a given duration (item ID + snooze hours) |
| `get_patterns` | Returns pattern observations for a given subject (context, task type, or global) |
| `find_similar_situations` | Searches closed contexts for situational matches to a given active context |
