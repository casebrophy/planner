# Ingestion Pipeline

Every data source → `RawInput` → 9-stage pipeline. Tier classification determines which model handles each stage.

## Source Interface

```go
// Source is implemented by every data source adapter.
type Source interface {
    // Name returns a stable identifier for this source type.
    // Used as source_type in raw_inputs. E.g. "email", "transaction_bank", "receipt".
    Name() string

    // DefaultTier returns the sensitivity tier for this source.
    // Can be overridden per-input by the classifier at stage 2.
    DefaultTier() SensitivityTier

    // Start begins receiving data, calling emit for each incoming item.
    // Should block until ctx is cancelled.
    Start(ctx context.Context, emit EmitFunc) error
}

// EmitFunc is called by an adapter when new data arrives.
type EmitFunc func(ctx context.Context, input RawInput) error

// RawInput is the normalised envelope every adapter produces.
type RawInput struct {
    SourceType  string            // matches Source.Name()
    RawContent  string            // full raw content, source-specific format
    Metadata    map[string]string // optional hints to the pipeline
    Tier        SensitivityTier   // if zero, DefaultTier() is used
    ReceivedAt  time.Time
}
```

## Sensitivity Tiers

| Tier | Rule | Examples |
|------|------|----------|
| 1 — API permitted | No PII in raw form | Receipts, task content, voice captures |
| 2 — Local then API | Contains PII, sanitize locally first | Bank CSV, emails, credit card exports |
| 3 — Fully local | Never leaves server | Health data, user-flagged |

Default tier by source table. Classifier can promote (never demote). Tier 2/3 permanent once assigned.

| Source | Default tier |
|--------|-------------|
| Receipt (photo/text) | 1 |
| Voice capture | 1 |
| Task / note entry | 1 |
| Email (general) | 2 |
| Email (financial) | 2 |
| Bank CSV | 2 |
| Credit card CSV | 2 |
| Apple Health export | 3 |
| User-flagged input | 3 |

## Pipeline Stages

1. **Store raw input** — write to `raw_inputs` with `status = pending` before any AI processing; non-negotiable first step
2. **Classify sensitivity tier** — start from `DefaultTier()`, run local PII pattern classifier, promote Tier 1 → 2 if indicators found
3. **Extract structured data** — model router selects inferencer by tier; Tier 1 uses external API, Tier 2 uses local model to sanitize first, Tier 3 uses local model only
4. **Sanitize & promote (Tier 2 only)** — log promotion in `sanitization_log`; block if PII re-detected in sanitized output; after promotion treat as Tier 1
5. **Classify & route to context** — match extraction to active contexts; auto-assign (high confidence), flag tentative (low confidence), store unlinked, or create new context
6. **Write entities** — write `context_events`, `tasks`, `emails`/`transactions`, `task_notes` based on extraction output
7. **Embed chunks** — chunk and embed with tier-appropriate model; Tier 3 always uses local embeddings regardless of config
8. **Update context summary** — rewrite `contexts.summary` using tier-appropriate inferencer; capped at ~500 words
9. **Mark processed** — set `raw_inputs.status = processed`, `processed_at = now`

## Extraction Schemas

**Email extraction schema:**
```json
{
  "summary": "Two-sentence summary, no PII",
  "sender_name": "string",
  "sender_domain": "string",
  "date": "ISO date",
  "action_items": ["string"],
  "deadlines": [{"description": "string", "date": "ISO date or null"}],
  "suggested_context_keywords": ["string"],
  "sentiment": "neutral | positive | urgent | negative",
  "pii_detected": true
}
```

**Transaction extraction schema:**
```json
{
  "clean_name": "Starbucks",
  "amount_cents": -450,
  "date": "2024-03-15",
  "category": "food_and_drink",
  "account_last_four": "4821",
  "suggested_context_keywords": ["string"],
  "notable": false,
  "notable_reason": null
}
```

## Source Adapters (v1)

- **SMTP receiver** — receives forwarded emails via `emersion/go-smtp`; default tier 2; financial institution domains confirmed at stage 2
- **CSV transaction importer** — bank/credit card CSV exports; always tier 2; supports Chase checking, Chase credit, Amex, Generic; deduplicates by `(source, date, description, amount)`; accepts `POST /api/v1/sources/transactions` or watched directory
- **Receipt importer** — photo or plain text; default tier 1; promotes to tier 2 if card numbers detected at stage 2

## Error Handling

- **Stage 1 failure** — input dropped, error logged; only unrecoverable failure
- **Stage 2–8 failures** — record stays `pending` or moves to `failed`; retry queue with exponential backoff capped at 1 hour
- **Local model unavailable** — Tier 2/3 queues and waits; never falls back to external API
- **External API unavailable** — Tier 1 queues and retries; Tier 2/3 unaffected
- **Promotion blocked by PII re-detection** — flagged for manual review in frontend; never silently promoted or dropped
