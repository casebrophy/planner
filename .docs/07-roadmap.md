# Build roadmap

**Guiding principle:** Build the smallest thing that's useful, then extend it. Each phase produces something you can actually use.

## Phase 1 — Working core
**Goal:** Tasks work end-to-end via Claude and voice — capture, view, complete.
**Deliverables:**
- Go backend with MCP server and REST API
- SQLite schema (tasks, notes, tags)
- Docker Compose for the backend
- SKILL.md for Claude task detection
**Done when:** Siri shortcut + Claude + MCP + backend pipeline works reliably for basic task capture and retrieval.

---

## Phase 2 — Contexts
**Goal:** Group related things together and let Claude reason across contexts.
**Deliverables:**
- `contexts`, `context_events` tables; `context_id` FK on `tasks`
- MCP tools: `create_context`, `get_context`, `list_contexts`, `update_context`, `link_task_to_context`
- SKILL.md: context detection and cross-context query handling
**Done when:** Context creation, linking, and querying work reliably through conversation.

## Phase 3 — Email ingestion
**Goal:** Forward an email and have the system extract tasks and update relevant contexts automatically.
**Deliverables:**
- SMTP receiver, email parser, ingestion processing loop
- Claude extraction prompt for emails
- `raw_inputs`, `emails` tables
- SMTP container, MX record, DNS/port setup
**Done when:** Forwarding a real email produces a correctly extracted task and context update, consistently.

---

## Phase 3b — Clarification queue
**Goal:** System accumulates unresolvable questions and surfaces them as a swipeable review deck — no push notifications, no interruptions.
**Deliverables:**
- Clarification item generator wired into ingestion and context engine
- `clarification_items` table; REST endpoints (`GET`, resolve, snooze)
- `ClarificationCard` + `ClarificationSession` shared components
- Task thread system (`task_thread_entries` table, `add_thread_update` MCP tool)
- Inactivity detection job; context debrief flow; `context_outcomes` table
- MCP tools: `get_clarification_queue`, `resolve_clarification`, `snooze_clarification`, `add_thread_update`, `get_thread`
- Triggers: low-confidence context match, ambiguous email action, auto-created context, stalled task, uncertain voice capture, context closure debrief (24h delay)
**Done when:** Queue fills naturally from email ingestion, cards are answerable in under 5 seconds, and resolution correctly updates underlying records.

---

## Phase 4 — Frontend (web shell)
**Goal:** Visual interface for reviewing and managing everything the system has captured, web-first.
**Deliverables:**
- Vue 3 + Vite + Pinia; vue-router with shell detection
- Views: Dashboard, Task board, Context board, Context detail, Task detail, Capture
- Shared component library built touch-first; web shell sidebar + multi-column layouts
**Done when:** Web shell gives a complete picture of the system and the shared component library is solid enough to build the mobile shell on top of.

---

## Phase 4b — Mobile shell (Capacitor)
**Goal:** The same Vue app packaged as a native iOS app with a capture-first mobile interface.
**Deliverables:**
- Mobile shell (bottom tab bar, full-screen views); Capture tab as default; Today tab
- Capacitor plugins: camera, photo library, haptics, share sheet
**Done when:** iOS app is installed and the receipt capture flow works end-to-end — photo to processed transaction — without touching the web app.

---

## Phase 5 — Transaction ingestion
**Goal:** Upload a bank export CSV and have transactions associated with contexts automatically.
**Deliverables:**
- CSV parser with per-bank format adapters
- AI model layer (`Inferencer`/`Embedder` interfaces, Anthropic + Ollama implementations, `ModelRouter`)
- Ollama container; sensitivity tier classification; sanitization/promotion gate
- `transactions`, `sanitization_log` tables
- Frontend: transaction review view; context detail with linked transactions
**Done when:** Uploading a real bank export produces correctly matched, sanitized transactions for at least 70% of rows, with no raw PII in extraction output.

---

## Phase 5b — Pattern recognition (Layer 1)
**Goal:** Statistical summaries over task and context data surface behavioural insights — no ML, just SQL aggregations Claude reasons over.
**Deliverables:**
- Statistical summary queries (completion rate, duration accuracy, overdue patterns, context lifetime)
- `outcome_observations`, `pattern_observations` tables (TTL-cached)
- Task completion debrief card; context closure debrief sequence (`ClosingReview`, 3–4 cards)
- `PatternInsight` shared component; "similar situations" section in context detail
- MCP tools: `get_patterns`, `find_similar_situations`, `get_outcome_observations`, `record_outcome`
- Inline duration/completion hints at task creation

**Prerequisite:** At least 4 weeks of real usage data.

**Done when:** At least one pattern surfaces per week that changes what you do in the next 48 hours.

---

## Phase 6 — Semantic search (RAG)
**Goal:** Claude can search your data by meaning, not just structure.
**Deliverables:**
- sqlite-vec extension; `OllamaEmbedder` implementation
- Embedding generation wired into ingestion pipeline
- `search_semantic` MCP tool with re-ranking heuristic
- SKILL.md additions: when to use semantic vs. structured search
- Indexed content: email summaries, context events, task notes/title/description, voice transcripts, context summaries
**Done when:** "Did I make any commitments this week?" works reliably, and Claude correctly chooses between semantic and structured queries.

---

## Phase 7 — Scheduling
**Goal:** Claude can propose a weekly schedule based on tasks, deadlines, and (eventually) calendar.
**Deliverables:**
- `time_blocks` table; scheduling MCP tools: `get_schedule`, `create_time_block`, `confirm_time_block`
- Duration estimation at task creation; schedule view (weekly calendar) in frontend
- Phase 7a: prioritised task order with time estimates, no calendar sync
- Phase 7b: iCal feed consumer; Claude proposes slots against real availability; confirmed blocks sync back
**Done when:** You use the scheduling feature at least once a week and find the proposals useful.

---

## Phase 8 — Intelligence layer (Python ML service)
**Goal:** Containerised Python service providing ML-powered analysis — pattern clustering, archetypes, situational matching (Layers 2 and 3 of pattern recognition).
**Deliverables:**
- HTTP API service (Go calls it; it never writes to DB — Go owns data, Python owns ML computation)
- Layer 2 (clustering and archetypes) and Layer 3 (situational matching) of pattern system
- Specific models, API contract, scheduling, and frontend surfacing to be designed once earlier phases are stable

**Prerequisite:** At least 2–3 months of real usage data across tasks, contexts, transactions, and emails.

**Done when:** Designed in a future session once the data layer and earlier phases are stable.

---

## Phase 9 — Intent framework
**Goal:** System decomposes high-level goals into executable plans using context and pattern data, presents for confirmation, and executes via pluggable adapters — new capabilities added through conversation, not code.
**Deliverables:**
- Intent recognition engine (embedding-based, adapter registry)
- Slot-filling engine with five fill strategies (context, transactions, preferences, recent activity, ask)
- `intent_adapters`, `intent_executions`, `workflow_creation_sessions` tables
- Conversational adapter creation flow (6-step guided conversation)
- Confirmation UI (bottom sheet mobile / inline card web); crystallisation logic after N consistent executions
- Automations management view (web + mobile)
- MCP tools: `recognise_intent`, `get_adapter`, `list_adapters`, `fill_slots`, `execute_intent`, `save_adapter`
- One reference Tier 3 adapter (grocery ordering) validated end-to-end
**Done when:** Full lifecycle works for at least one adapter — expression, slot filling, confirmation, execution, outcome capture, and crystallisation.

---

## Phase 9b — Adapter expansion
**Goal:** Extend the framework to additional domains through the Tier 2 creation flow — no new infrastructure.
**Deliverables:**
- New adapters defined conversationally: express intent → creation flow → slots/fill strategies/execution spec → save → crystallise after 5 consistent executions → promote to Tier 3 if warranted
**Done when:** System handles at least 5 distinct intent types reliably, with at least 3 added via creation flow without developer involvement.

---

## Deferred (not on roadmap yet)

- **Receipt capture** — photo → transaction; adds OCR complexity
- **Apple Health import** — useful for health contexts; straightforward once pipeline exists
- **Notifications** — push when something important arrives; requires notification infrastructure decision
- **Mobile-optimised frontend** — Vue app usable on mobile but not optimised; native app is a larger investment
- **Multi-source deduplication** — not a problem until two sources produce the same data

## What not to build

- **User accounts / auth** — single user, API key is sufficient
- **Team features** — personal tool; collaboration requires fundamental redesign
- **Webhook integrations** — data sources only, not productivity app connections
- **Real-time sync** — frontend polls; WebSocket adds complexity not worth it for a personal tool
