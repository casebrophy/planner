# Build roadmap

**Guiding principle:** Build the smallest thing that's useful, then extend it. Each phase produces something you can actually use.

> **Cross-cutting dependency: AI model layer.** The `Inferencer`/`Embedder`/`ModelRouter` interfaces (designed in `08-ai-model-layer.md`) are prerequisites for Phases 5 (AI enrichment), 6 (RAG), and 8 (ML service). The current `ClaudeCodeExtractor` in `ingestbus/extractor/` is a narrow, pipeline-specific stopgap — it does not implement the designed interfaces. Building the model layer is a prerequisite for any phase that needs local inference, sensitivity-tier routing, or embeddings.

---

## Phase 1 — Working core  ✅ Complete
**Goal:** Tasks work end-to-end via Claude and voice — capture, view, complete.
**Deliverables:**
- ~~Go backend with MCP server and REST API~~ done
- ~~PostgreSQL schema (tasks, tags, contexts)~~ done
- ~~Docker Compose for the backend~~ done
- ~~SKILL.md for Claude task detection~~ done
**Ship when:** Backend starts, MCP tools respond, tasks round-trip through create → query → complete.
**Success when:** Siri shortcut + Claude + MCP + backend pipeline works reliably for basic task capture and retrieval.

---

## Phase 2 — Contexts  ✅ Complete
**Goal:** Group related things together and let Claude reason across contexts.
**Deliverables:**
- ~~`contexts`, `context_events` tables; `context_id` FK on `tasks`~~ done
- ~~MCP tools: `create_context`, `get_context`, `list_contexts`, `update_context`, `link_task_to_context`~~ done
- ~~SKILL.md: context detection and cross-context query handling~~ done
**Ship when:** All context MCP tools respond; contexts link to tasks; events append correctly.
**Success when:** Context creation, linking, and querying work reliably through conversation.

---

## Phase 3 — Email ingestion  ⚠️ Partial
**Goal:** Forward an email and have the system extract tasks and update relevant contexts automatically.
**Deliverables:**
- ~~`raw_inputs`, `emails` tables~~ done
- ~~Read-only API routes (query emails, query/reprocess raw inputs)~~ done
- ~~SMTP receiver (`smtpbus`), email parser + ingestion pipeline (`ingestbus`)~~ built and wired; disabled by default (`PLANNER_SMTP_ENABLED=false`)
- ~~Wire `ingestbus.Reprocess()` into `rawinputapp` reprocess endpoint~~ done
- Claude extraction prompt for emails
- SMTP container, MX record, DNS/port setup
**Ship when:** All code deliverables above are done; SMTP enabled in Docker; extraction prompt committed.
**Success when:** Forwarding a real email produces a correctly extracted task and context update, consistently.

**Remaining:** Claude extraction prompt tuning, SMTP container + MX/DNS production setup.

---

## Phase 3b — Clarification queue  ⚠️ Partial
**Goal:** System accumulates unresolvable questions and surfaces them as a swipeable review deck — no push notifications, no interruptions.
**Deliverables:**
- ~~`clarification_items` table~~ done
- ~~Unified thread system (`thread_entries` table with subject_type/subject_id, `add_thread_update` MCP tool)~~ done (data layer + routes)
- ~~`inactivity_checks` table~~ done
- ~~`outcome_observations` table; `debrief_status`/`outcome` columns on tasks and contexts~~ done
- ~~Clarification REST endpoints (query, resolve, snooze, dismiss, count)~~ done
- ~~Clarification item generator wired into ingestion and context engine~~ done (wired into ingestbus — generates `new_context`, `context_assignment`, `ambiguous_action`, `ambiguous_deadline`)
- ~~`ClarificationCard` + `ClarificationSession` shared components~~ done
- ~~Inactivity detection job; context debrief flow~~ done (inactivitybus wired as scheduled goroutine in main.go)
- ~~MCP tools: `get_clarification_queue`, `resolve_clarification`, `snooze_clarification`~~ done
- Remaining card type generators — moved to Phase 4c
**Ship when:** All data layer, REST, MCP, and frontend components respond. At least the 4 email-triggered card types generate.
**Success when:** Queue fills naturally from email ingestion, cards are answerable in under 5 seconds, and resolution correctly updates underlying records.

**Remaining:** `voice_reference` card generator — unblocked now that voice pipeline exists; needs wiring into ingestbus extractor.

---

## Phase 4 — Frontend (web shell)  ✅ Complete
**Goal:** Visual interface for reviewing and managing everything the system has captured, web-first.
**Deliverables:**
- ~~Vue 3 + Vite + Pinia; vue-router with shell detection~~ done
- ~~Views: Dashboard, Task board, Context board, Context detail, Task detail, Capture~~ done (Clarification, Today, Search, Settings views also implemented)
- ~~Shared component library built touch-first; web shell sidebar + multi-column layouts~~ done
**Ship when:** All views render, API calls succeed, component library covers all current domains.
**Success when:** Web shell gives a complete picture of the system and the shared component library is solid enough to build the mobile shell on top of.

---

## Phase 4b — Mobile shell (PWA + Capacitor)  ⚠️ Partial (PWA done, Capacitor deferred)
**Goal:** The same Vue app as a mobile-optimised experience — PWA first, native iOS later.
**Deliverables:**
- ~~PWA shell: manifest, service worker (workbox/autoUpdate), responsive layout, iOS meta tags~~ done
- ~~`useShell()` composable for mobile/desktop detection; `MobileTabBar` (5-tab) + responsive `AppShell`~~ done
- ~~Dynamic default route: `/capture` on mobile, `/dashboard` on desktop~~ done
- ~~Go static server `Cache-Control: no-cache` for `sw.js` and `manifest.json`~~ done
- Capacitor native packaging (iOS): `@capacitor/core`, `cap init`, `cap sync` — deferred
- Capacitor plugins: camera, photo library, haptics, share sheet — deferred
**Ship when:** PWA installs on iOS, responsive layout works, capture flow is usable via browser.
**Success when:** iOS app is installed and the receipt capture flow works end-to-end — photo to processed transaction — without touching the web app.

**Remaining:** Capacitor native packaging deferred pending decision on whether PWA is sufficient.

---

## Phase 4c — Feature completeness  ⚠️ Partial
**Goal:** Wire generators, MCP tools, and trigger logic to data infrastructure that already exists but isn't connected.

**Clarification queue completeness:**
- ~~Wire `overlapping_contexts` card generator (keyword/tag-based, 2+ shared tags)~~ done
- ~~Wire `context_debrief` cards (3-card sequence, snoozed 24h after context close)~~ done
- ~~Wire `task_debrief` cards (on completion with blockers or duration overrun >2x)~~ done
- ~~`inactivity_prompt` cards from inactivitybus~~ done (already existed)
- ~~Priority score computation from kind weights~~ done (already existed)
- `voice_reference` card generator — unblocked (voice pipeline exists), needs wiring into ingestbus
- `stale_task` — already handled by `inactivity_prompt` kind

**MCP tool gaps:**
- ~~`record_outcome`~~ done (already existed)
- ~~`get_outcome_observations` — query observations for a subject~~ done
- ~~`task_debrief` added to `get_clarification_queue` kind filter~~ done

**Thread enrichment:**
- ~~Optional `Extract` flag on `NewThreadEntry` + `Extractor` interface~~ done
- ~~`WithExtractor()` method on threadbus.Business~~ done
- Concrete thread extractor implementation — deferred (interface ready)
- Auto-generate clarification card when `requires_action = true` — deferred

**Debrief trigger logic:**
- ~~`debriefbus` package with `OnTaskCompleted()` and `OnContextClosed()`~~ done
- ~~Wired into MCP `complete_task`, `update_task`, `update_context` handlers via goroutines~~ done

**Ship when:** ~~All built card types have generators wired; MCP tools respond; thread extraction interface exists; debrief cards appear on task completion and context close.~~ Shipped.
**Success when:** The clarification queue fills naturally from real usage across all trigger types, not just email ingestion.

---

## Phase 5 — Transaction ingestion  ⚠️ Partial (CRUD done, AI enrichment deferred)
**Goal:** Upload a bank export CSV and have transactions stored and reviewable.
**Deliverables:**
- ~~CSV parser with per-bank format adapters~~ done
- ~~`transactions` table~~ done
- ~~REST API endpoints (CRUD + CSV import)~~ done
- ~~Frontend: transaction board view with import and review~~ done
- AI model layer (`Inferencer`/`Embedder` interfaces, `ModelRouter`) — deferred to AI model layer prerequisite
- Ollama container; sensitivity tier classification; sanitization/promotion gate — deferred
- `sanitization_log` table — deferred
- Transaction import through full ingest pipeline (context routing, clarification, embedding) — deferred
**Ship when:** CSV import stores transactions; frontend displays and filters them; manual context assignment works.
**Success when:** Uploading a real bank export produces correctly matched, sanitized transactions for at least 70% of rows, with no raw PII in extraction output. (Requires AI model layer.)

---

## Phase 5b — Pattern recognition (Layer 1)
**Goal:** Statistical summaries over task and context data surface behavioural insights — no ML, just SQL aggregations Claude reasons over.
**Deliverables:**
- Statistical summary queries (completion rate, duration accuracy, overdue patterns, context lifetime)
- ~~`outcome_observations` table~~ done (created early for thread/debrief support); `pattern_observations` table (TTL-cached)
- Task completion debrief card; context closure debrief sequence (`ClosingReview`, 3–4 cards)
- `PatternInsight` shared component; "similar situations" section in context detail
- MCP tools: `get_patterns`, `find_similar_situations`, `get_outcome_observations`, `record_outcome`
- Inline duration/completion hints at task creation

**Prerequisite:** At least 4 weeks of real usage data. Phase 4c (debrief triggers + outcome MCP tools) should be done first.

**Ship when:** SQL aggregation queries exist; `pattern_observations` table migrated; MCP tools respond; PatternInsight component renders.
**Success when:** At least one pattern surfaces per week that changes what you do in the next 48 hours.

---

## Phase 6 — Semantic search (RAG)
**Goal:** Claude can search your data by meaning, not just structure.
**Deliverables:**
- pgvector extension; `OllamaEmbedder` implementation
- Embedding generation wired into ingestion pipeline (stage 7)
- Automatic context summary rewrite on new events (stage 8)
- `search_semantic` MCP tool with re-ranking heuristic
- SKILL.md additions: when to use semantic vs. structured search
- Indexed content: email summaries, context events, task notes/title/description, voice transcripts, context summaries

**Prerequisite:** AI model layer (`Inferencer`/`Embedder`/`ModelRouter` interfaces).

**Ship when:** Embeddings table exists; pipeline stages 7-8 execute; `search_semantic` MCP tool returns relevant results.
**Success when:** "Did I make any commitments this week?" works reliably, and Claude correctly chooses between semantic and structured queries.

---

## Phase 7a — Daily Planner  ⚠️ Partial
**Goal:** AI-generated daily task plan with smart grouping, plus events as fixed commitments that constrain the plan.
**Deliverables:**
- ~~`events` table — fixed commitments (appointments, trips) with optional location; created via voice ingest or manually~~ done
- ~~REST endpoints: CRUD for events~~ done (GET/POST/PUT/DELETE /api/v1/events)
- ~~MCP tools: `create_event`, `list_events`, `get_event`, `update_event`, `delete_event`~~ done
- ~~Voice ingest update: Claude classifies input as task vs. event~~ done (extractor returns Events array, ingestbus creates events)
- ~~`daily_plans` + `daily_plan_items` tables — AI-generated grouped task list with override tracking~~ done
- ~~REST endpoints: get/generate daily plan, update plan items (reorder, dismiss, complete)~~ done
- ~~Morning batch job (configurable, default 7am) generates daily plan; on-demand regeneration via API~~ done
- ~~AI duration estimation for tasks missing `duration_min`; stored as `ai_duration_min` on plan items~~ done
- ~~MCP tools: `get_daily_plan`, `generate_daily_plan`~~ done
- ~~Frontend: daily plan view with grouped task cards, drag-reorder, dismiss actions; event list/create~~ done
- Event-task implication reasoning: Claude surfaces prerequisite relationships (e.g. "change wipers before road trip") via clarification system
- User interactions captured for training data: drag reorder (`user_position`), duration override (`user_duration_min`), dismiss with structured reason + freeform note
**Ship when:** Daily plan generates from open tasks + events; plan view renders with drag reorder and dismiss.
**Success when:** Morning plan is useful enough to check daily; dismiss reasons capture why AI got it wrong.

**Remaining:** Morning batch job (goroutine is wired but untested in production), event-task implication reasoning via clarification system.

---

## Phase 7b — Calendar View + Time Blocks  ✅ Complete
**Goal:** Self-contained weekly calendar view showing events and scheduled task blocks — the planner is the calendar.
**Deliverables:**
- ~~`time_blocks` table — time-slotted task scheduling (any task, not just daily plan items)~~ done
- ~~Weekly calendar view showing events (with times) + time blocks side by side~~ done
- ~~Manual time block creation (assign any task to a time slot)~~ done
- ~~REST API: CRUD for time blocks + schedule query (events + blocks merged)~~ done (5 time block routes + 1 schedule query route)
- ~~MCP tools: `get_schedule`, `create_time_block`, `confirm_time_block`~~ done
- ~~15-min configurable buffer between tasks~~ deferred — ordering matters more than precise time slots; buffer logic makes more sense once auto-scheduling is mature
**Ship when:** Weekly calendar view renders events + time blocks; manual scheduling works; MCP tools respond.
**Success when:** You use the planner as your primary calendar.

---

## Phase 7b.1 — Task & Context Model Refinement  ✅ Complete
**Goal:** Simplify task lifecycle, add task dependencies with auto-blocking, and clarify the project-vs-area distinction for contexts.
**Deliverables:**
- ~~Task statuses simplified: `open`, `blocked`, `done`, `dismissed` (removed `in_progress`, renamed `todo`→`open`, `cancelled`→`dismissed`)~~ done
- ~~Task dependencies via `task_dependencies` junction table with auto-blocking/unblocking~~ done
- ~~`blocked_reason` field for manual blocks~~ done
- ~~Context `kind` field: `project` (time-bounded, closeable) vs `area` (ongoing, never closes)~~ done
- ~~Cascade dismiss: closing a project context dismisses all its open/blocked tasks~~ done
- ~~REST API: 4 dependency endpoints, kind on context CRUD, blockedReason on task CRUD~~ done
- ~~MCP tools: `add_task_dependency`, `remove_task_dependency`, `get_task_dependencies`, kind on `create_context`, blocked_reason on `create_task`~~ done
- ~~Frontend: updated status display, context kind support, blocked task styling~~ done
**Ship when:** All layers updated, build clean, frontend renders new statuses and kind.
**Success when:** Task dependencies surface in the daily plan, and closing a project context cleanly wraps up its tasks.

---

## Phase 7c — Life Dashboard Primitives  ⚠️ Partial
**Goal:** Extend the planner from task/calendar tool into a single surface for daily life. Three new primitives: notes (knowledge capture), recurring tasks (habits/routines), and trackable logs (history over time). Plus retroactive classification for existing data.

**Design philosophy:** Avoid domain-specific tables (health, people, media, etc.). Instead, notes + tags + recurring tasks + trackable logs + contexts compose to cover any life domain. The "life dashboard" emerges from tagging and querying, not from custom tables per domain.

**Notes / Knowledge capture:**
- ~~`notes` table + `note_tags` junction (reuses existing `tags` table)~~ done
- ~~Notes get optional `context_id` — links knowledge to situations (e.g. PT phone number lives in "Physical Therapy" context)~~ done
- ~~REST API: CRUD for notes, tag management on notes, query by tag~~ done
- ~~Voice/text ingest learns a third classification: `note` (alongside task and event)~~ done
- ~~Auto-tagging: extractor suggests 1-3 tags per note, creates new tags as needed~~ done
- ~~MCP tools: `create_note`, `search_notes` (keyword + tag filter), `list_notes_by_tag`~~ done
- Frontend: notes view with tag filtering, tag management
- Clarification card when classifier is low-confidence on task vs. note distinction

**Recurring tasks:**
- ~~Add `recurrence_rule` (nullable text, e.g. `FREQ=DAILY`, `FREQ=WEEKLY;BYDAY=TH`) and `recurrence_parent_id` (nullable FK to tasks) on existing `tasks` table~~ done
- On completion of a recurring task, system auto-creates next instance: same title/priority/context/tags, new `due_date` per rule, reference back to parent
- Recurring tasks appear naturally in daily plan generation
- Frontend: recurrence indicator on task cards, recurrence rule input on task form

**Trackable logs:**
- ~~`activity_logs` table — generic log entries: `(log_id, subject_type, subject_id, value, logged_at)` where subject is a note, task, or any entity~~ done
- ~~`activitylogbus` + `activitylogapp` wired in~~ done
- Any note or recurring task can be "trackable" — each completion or manual check-in creates a log entry
- Queryable: streak count, frequency, last occurrence, count over period
- Frontend: streak/frequency display on trackable items, simple log history view

**Retroactive classification:**
- ~~REST endpoint: `POST /api/v1/classify` — batch classification via `classifyapp`~~ done
- Frontend classify button on task board

**Ship when:** Notes with auto-tagging work via voice/text. Recurring tasks generate next instance on completion. Trackable items show streak/frequency. Classify button organizes orphan tasks.
**Success when:** You can photograph a PT handout → system extracts appointments (events), exercises (notes tagged "physical-therapy"), phone number (note) → later ask "what's my PT's phone number?" and get an answer. Recurring daily tasks show up in the morning plan. Habit streaks are visible.

**Remaining:** Recurring task auto-spawn on completion, streak/frequency frontend display, notes frontend view, classify button in UI.

**Future enhancement:** Phase 6 (semantic search) makes recall dramatically better — "what do I know about X" becomes meaning-based, not keyword-based.

---

## Phase 7d — Entity Links  ⚠️ In Progress
**Goal:** First-class relationships between any two entities (task↔note, task↔event, note↔note, etc.) — bidirectional, typed, and surfaced in detail views as a "Related Items" panel.

**Deliverables:**
- `entity_links` table (v1.20 migration) — `(link_id, source_type, source_id, target_type, target_id, link_kind, created_at)`
- `entitylinkbus` — business layer (model, Storer interface, CRUD methods)
- `entitylinkdb` — store implementation + integration tests
- `entitylinkapp` — HTTP handlers + routes (`POST /api/v1/entity-links`, `GET /api/v1/entity-links?source_type=&source_id=`, `DELETE /api/v1/entity-links/:id`)
- Wire into `main.go` + `dbtest`
- `entity_link` clarification kind + `EntityLinkOptions` (surfaces low-confidence relationship suggestions from ingest)
- `classifyapp` extended: classify notes + events via `entity_type` param
- Auto-classify notes + events on create (async goroutine in app handler)
- Frontend: TypeScript types + `entityLinkService` + `entityLinkStore`
- Frontend: "Related Items" panel in `NoteDetailView` and `TaskDetailView`
- Frontend: `ClarificationCard` for `entity_link` kind + resolution side-effect

**Ship when:** All layers wired; related items panel renders; auto-linking fires on note/event create.
**Success when:** Creating a note about a task automatically surfaces a link suggestion; resolving it adds the relation and it appears in both detail views.

---

## Phase 8 — Intelligence layer (Python ML service)
**Goal:** Containerised Python service providing ML-powered analysis — pattern clustering, archetypes, situational matching (Layers 2 and 3 of pattern recognition).
**Deliverables:**
- HTTP API service (Go calls it; it never writes to DB — Go owns data, Python owns ML computation)
- Layer 2 (clustering and archetypes) and Layer 3 (situational matching) of pattern system
- Specific models, API contract, scheduling, and frontend surfacing to be designed once earlier phases are stable

**Prerequisite:** At least 2–3 months of real usage data across tasks, contexts, transactions, and emails.

**Ship when:** ML service container starts; Go can call its HTTP API; at least one layer produces results.
**Success when:** Designed in a future session once the data layer and earlier phases are stable.

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
**Ship when:** Tables migrated; at least one adapter works end-to-end through the full lifecycle.
**Success when:** Full lifecycle works for at least one adapter — expression, slot filling, confirmation, execution, outcome capture, and crystallisation.

---

## Phase 9b — Adapter expansion
**Goal:** Extend the framework to additional domains through the Tier 2 creation flow — no new infrastructure.
**Deliverables:**
- New adapters defined conversationally: express intent → creation flow → slots/fill strategies/execution spec → save → crystallise after 5 consistent executions → promote to Tier 3 if warranted
**Ship when:** At least 3 adapters created via conversation flow without code changes.
**Success when:** System handles at least 5 distinct intent types reliably, with at least 3 added via creation flow without developer involvement.

---

## Deferred (designed but not scheduled)

- **Receipt capture** — photo → transaction; adds OCR complexity
- **Apple Health import** — useful for health contexts; straightforward once pipeline exists
- **Notifications** — push when something important arrives; requires notification infrastructure decision
- ~~**Mobile-optimised frontend**~~ done (PWA with responsive shell, bottom tab bar on mobile)
- **Multi-source deduplication** — not a problem until two sources produce the same data
- **`Source` interface abstraction** — generic adapter pattern (`Source`/`EmitFunc` from `04-ingestion-pipeline.md`); current adapters are wired ad-hoc
- **Pipeline retry queue** — exponential backoff for failed ingestion runs; currently failures just set `status=failed`
- **`sanitization_log` table** — tracks Tier 2 PII promotion decisions; designed in `04-ingestion-pipeline.md`
- **`useSessionStore` / `X-Session-ID`** — frontend session tracking designed in `09-frontend.md`; no phase references it
- ~~**Backup automation**~~ done (`zarf/deploy/backup.sh` + systemd timer, 7-day retention)
- **Capacitor native iOS** — deferred from Phase 4b pending PWA evaluation

## What not to build

- **User accounts / auth** — single user, API key is sufficient
- **Team features** — personal tool; collaboration requires fundamental redesign
- **Webhook integrations** — data sources only, not productivity app connections
- **Real-time sync** — frontend polls; WebSocket adds complexity not worth it for a personal tool
