# Ollama Fallback Design

**Date:** 2026-04-03  
**Scope:** Add local Ollama model as fallback extraction path when Claude API fails with rate limits or context limits

## Problem Statement

Classification currently relies solely on Claude Code CLI (Haiku → Sonnet/Opus escalation via confidence thresholds). If Claude API becomes unavailable due to rate limits (429), context window exhaustion, or API downtime, the entire extraction pipeline fails and classification stops working.

The goal is to add a **parallel fallback path** using a locally-hosted Ollama model that activates only when Claude hits specific failure modes, allowing classification to continue functioning.

## Design Goals

1. **Transparent fallback** — classifyapp doesn't change; it uses the same `Extractor` interface
2. **Claude-first** — prefer Claude for its reasoning capability; Ollama only when Claude fails
3. **Specific trigger conditions** — fallback only on rate limits (429), context limits, API downtime; not on other errors
4. **Flexible model choice** — Ollama supports various models; user configures which one
5. **No confidence escalation** — Ollama is a fallback, not part of the Claude confidence chain
6. **Easy to test** — each extractor (Claude, Ollama, Failover) can be tested independently

## Architecture

### Two Independent Extraction Paths

```
FailoverExtractor (implements Extractor interface)
├─ Primary Path: ClaudeCodeExtractor
│  └─ On specific errors (429, context limit, API down) → trigger fallback
│  └─ On other errors → escalate through confidence chain (Sonnet/Opus)
│  └─ On success → return result
│
└─ Fallback Path: OllamaExtractor
   └─ HTTP client to local Ollama instance
   └─ Structured output matching Claude schema
   └─ Single attempt (no escalation)
   └─ Returns result or error
```

### Component: OllamaExtractor

**File:** `business/domain/ingestbus/extractor/ollama.go`

Implements `Extractor` interface with HTTP client to Ollama endpoint.

**Constructor:**
```go
NewOllamaExtractor(ollamaURL string) *OllamaExtractor
```

**Methods:**
- `ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts) (EmailExtraction, error)`
- `ExtractText(ctx, text, activeContexts) (TextExtraction, error)`

**Implementation details:**
- POST to `{ollamaURL}/api/generate` (or appropriate Ollama endpoint)
- Use same extraction prompts as Claude (reuse `prompt.go` prompt builders)
- Request structured output matching `emailExtractionSchema` and `textExtractionSchema`
- Parse response JSON into `EmailExtraction` or `TextExtraction`
- Confidence scores: set to 0.85 for successful extractions (high but not 1.0, to indicate local model)
- Timeout: 30 seconds per request
- Context limit handling: if Ollama hits context limit, log error and return gracefully

### Component: FailoverExtractor

**File:** `business/domain/ingestbus/extractor/failover.go`

Wraps both extractors and implements fallback logic.

**Constructor:**
```go
NewFailoverExtractor(claudeExtractor *ClaudeCodeExtractor, ollamaExtractor *OllamaExtractor) *FailoverExtractor
```

**Methods:**
- `ExtractEmail(ctx, subject, bodyText, fromAddress, activeContexts) (EmailExtraction, error)`
- `ExtractText(ctx, text, activeContexts) (TextExtraction, error)`

**Fallback Logic (applies to both methods):**

1. Try Claude path
2. If error matches a fallback trigger:
   - Log the error and fallback activation
   - Try Ollama path
   - Return Ollama result (success or error)
3. If error doesn't match fallback triggers:
   - Return error as-is (let Claude confidence escalation handle it)
4. If success:
   - Return Claude result

**Fallback Trigger Conditions:**

Error matches if:
- HTTP 429 (rate limit)
- Error message contains `"context"` AND `"limit"` (context window exhaustion)
- Error message contains `"connection"` OR `"timeout"` OR `"refused"` (API unavailable)
- Error type indicates connection failure (network error)

**Non-triggering Errors:**

These errors escalate through Claude's confidence chain, not to Ollama:
- 400 Bad Request (malformed input)
- 401 Unauthorized (API key issue)
- 500 Internal Server Error (Claude API bug, will retry and escalate)
- Validation errors (invalid JSON, schema mismatch)

### Configuration

**Environment variables:**
- `OLLAMA_URL` — Ollama endpoint (e.g., `http://localhost:11434`). If empty, Ollama is disabled.
- `OLLAMA_ENABLED` — Boolean flag (optional, defaults to true if OLLAMA_URL is set)

**In `app/sdk/mux/config.go`:**
```go
OllamaURL     string
OllamaEnabled bool
```

**In `main.go` (api/services/planner/):**
```go
ollamaURL := os.Getenv("OLLAMA_URL")
ollamaEnabled := ollamaURL != ""
if os.Getenv("OLLAMA_ENABLED") == "false" {
  ollamaEnabled = false
}

var extractor extractor.Extractor
if ollamaEnabled {
  ollamaExt := extractor.NewOllamaExtractor(ollamaURL)
  extractor = extractor.NewFailoverExtractor(claudeExt, ollamaExt)
} else {
  extractor = claudeExt
}
```

### Integration Points

**Minimal changes to existing code:**

- `classifyapp/route.go` — Replace `extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)` with the failover setup (see config section above)
- No changes to `classifyapp.classify()` or `app/domain/*app` handlers
- No changes to `Extractor` interface

### Prompts

Reuse existing prompt builders from `business/domain/ingestbus/extractor/prompt.go`:
- `BuildEmailExtractionPrompt()` — for Ollama email extraction
- `BuildTextExtractionPrompt()` — for Ollama text/voice extraction

Ollama responses are expected to conform to the same JSON schemas as Claude (`emailExtractionSchema`, `textExtractionSchema`).

## Error Handling

**When Ollama fails:**
- Log the error with context (task/email being classified, error type)
- Return error to caller (classifyapp will handle — it already has error handling for extraction failures)
- No infinite retry loops or circular fallbacks

**When both Claude and Ollama fail:**
- Caller receives an error; classification skips that task/email
- User sees clarification card or error in UI (existing behavior)

**Logging:**
- Log each fallback activation (e.g., "Claude 429, falling back to Ollama")
- Log successful fallback (e.g., "Ollama extraction succeeded")
- Log Ollama errors (including timeout, connection refused, context limit)

## Testing Strategy

**Unit Tests:**

1. `extractor/ollama_test.go`
   - Mock HTTP client or use httptest server
   - Test successful extraction (both email and text)
   - Test Ollama error responses (timeout, 500, malformed JSON)
   - Test schema validation (response matches EmailExtraction/TextExtraction)

2. `extractor/failover_test.go`
   - Mock both Claude and Ollama extractors
   - Test fallback triggers on each error type (429, context limit, connection error)
   - Test non-triggering errors pass through
   - Test successful Claude path (no fallback needed)
   - Test successful Ollama fallback

**Integration Tests (Optional):**
- Spin up real Ollama instance in CI
- Test end-to-end extraction with fallback

**Manual Testing:**
- Set `OLLAMA_URL=http://localhost:11434` locally
- Run classify endpoint, verify fallback activates on 429 error

## Data Flow Diagram

```
classifyapp.classify()
├─ Query open tasks (no context)
├─ For each task:
│  └─ Call failoverExtractor.ExtractText()
│     ├─ Try ClaudeCodeExtractor
│     │  ├─ Success → return result, mark task classified
│     │  ├─ 429/context/API down error → trigger fallback
│     │  │  └─ Try OllamaExtractor
│     │  │     ├─ Success → return result, mark task classified
│     │  │     └─ Error → log, return error, skip task
│     │  └─ Other error → return error (confidence escalation handles)
│     └─ Error → skip task, maybe create clarification card
├─ Return count of classified tasks and clarifications created
```

## Future Extensibility

**Adding more fallback models:**
- Create new extractor (e.g., `GeminiExtractor` for Google's Gemini API)
- Implement `Extractor` interface
- Extend `FailoverExtractor` to support multiple fallbacks with priority order:
  ```go
  type FailoverExtractor struct {
    primary    Extractor
    fallbacks  []Extractor  // ordered by preference
  }
  ```
- Existing code continues to work unchanged

## Rollout

1. Deploy with `OLLAMA_ENABLED=false` by default (or omit `OLLAMA_URL`)
2. Once verified locally, set `OLLAMA_URL=http://localhost:11434` in production `.env`
3. Monitor logs for fallback activations and success rates
4. Adjust model choice or prompt if needed

## Success Criteria

- Classification continues to work when Claude hits rate limits
- Ollama fallback is transparent to classifyapp and API consumers
- Confidence scores and clarifications are still created (no silent failures)
- Logs clearly indicate when fallback is triggered and whether it succeeds
- No performance regression on the happy path (Claude success)
