# TOC

Lookup index for `.docs/` planning files. Skills resolve `file#section` references by reading the target file and extracting content under the matching `##` heading through the next heading of equal or higher level.

**At-a-glance brief:** `00-council-brief.md` — one-page summary of what the system is, current phase status, and design rules of thumb for advisors.

## By Domain
- task: `03-data-model.md#tasks`, `07-roadmap.md#phase-1--working-core`
- context: `03-data-model.md#contexts`, `05-context-engine.md#context-operations`, `05-context-engine.md#context-lifecycle`, `07-roadmap.md#phase-2--contexts`, `arch/context-backend.md`
- email: `03-data-model.md#emails`, `04-ingestion-pipeline.md#source-adapters-v1`, `07-roadmap.md#phase-3--email-ingestion`, `arch/email-backend.md`
- raw_input: `03-data-model.md#raw-inputs`, `07-roadmap.md#phase-3--email-ingestion`, `arch/rawinput-backend.md`
- transaction: `03-data-model.md#transactions`, `07-roadmap.md#phase-5--transaction-ingestion`, `arch/transaction-backend.md`
- tag: `03-data-model.md#tags`, `arch/tag-backend.md`
- debrief: `07-roadmap.md#phase-4c--feature-completeness`, `11-feedback-loop.md#feedback-loop`, `arch/debrief-backend.md`
- inactivity: `11-feedback-loop.md#inactivity-detection`, `arch/inactivity-backend.md`
- thread: `03-data-model.md#thread-entries`, `11-feedback-loop.md#task-threads`, `arch/thread-backend.md`
- clarification: `03-data-model.md#clarification-items`, `10-clarification-patterns.md#clarification-queue`, `07-roadmap.md#phase-3b--clarification-queue`, `arch/clarification-backend.md`
- observation: `03-data-model.md#outcome-observations`, `11-feedback-loop.md#outcome-observations`, `arch/observation-backend.md`
- inactivity_check: `03-data-model.md#inactivity-checks`, `11-feedback-loop.md#inactivity-detection`
- mcp: `arch/mcp-backend.md`
- check: `arch/check-backend.md`
- ingest: `arch/ingest-backend.md`
- smtp: `arch/smtp-backend.md`
- pattern: `10-clarification-patterns.md#pattern-recognition`, `11-feedback-loop.md#feedback-loop`, `07-roadmap.md#phase-5b--pattern-recognition-layer-1`
- intent: `12-intent-framework.md#intent-lifecycle`, `12-intent-framework.md#three-tier-adapters`, `07-roadmap.md#phase-9--intent-framework`
- scheduling: `05-context-engine.md#scheduling`, `05-context-engine.md#daily-plan-phase-7a`, `07-roadmap.md#phase-7a--daily-planner`, `07-roadmap.md#phase-7b--calendar-view--time-blocks`
- events: `03-data-model.md#events`, `05-context-engine.md#scheduling`, `arch/event-backend.md`
- voice_ingest: `arch/voiceingest-backend.md`
- timeblock: `03-data-model.md#time-blocks`, `07-roadmap.md#phase-7b--calendar-view--time-blocks`, `arch/timeblock-backend.md`
- schedule: `07-roadmap.md#phase-7b--calendar-view--time-blocks`, `arch/schedule-backend.md`
- server: `06-infrastructure.md#monitoring`, `arch/server-backend.md`
- dailyplan: `03-data-model.md#daily-plans`, `03-data-model.md#daily-plan-items`, `05-context-engine.md#daily-plan-phase-7a`, `arch/dailyplan-backend.md`
- notes: `03-data-model.md#notes`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- recurring-tasks: `03-data-model.md#tasks`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- activity-logs: `03-data-model.md#activity-logs`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- frontend: `09-frontend.md#navigation-structure`, `09-frontend.md#shared-components`, `09-frontend.md#routes`, `09-frontend.md#pwa`, `07-roadmap.md#phase-4--frontend-web-shell`
- pwa: `09-frontend.md#pwa`, `07-roadmap.md#phase-4b--mobile-shell-capacitor`
- entity-link: `03-data-model.md#entity-links`, `07-roadmap.md#phase-7d--entity-links`, `arch/entitylink-backend.md`, `arch/entitylink-frontend.md`
- embedding: `03-data-model.md#embeddings`, `07-roadmap.md#phase-6--semantic-search-rag`, `arch/embedding-backend.md`, `08-ai-model-layer.md#vector-storage-ddl`
- classification-correction: `03-data-model.md#classification-corrections`, `arch/correction-backend.md`, `arch/correction-frontend.md`
- knowledge-gap: `03-data-model.md#clarification-items`, `arch/knowledgegap-backend.md`
- transaction-split: `03-data-model.md#transaction-splits`, `arch/split-backend.md`
- note: `03-data-model.md#notes`, `07-roadmap.md#phase-7c--life-dashboard-primitives`, `arch/note-backend.md`, `arch/note-frontend.md`
- reclassify: `arch/reclassify-backend.md`, `arch/reclassify-frontend.md` (deprecated — being consolidated into correction; see beads planner-ow7c)

## By Concept
- sensitivity-tiers: `04-ingestion-pipeline.md#sensitivity-tiers`
- model-routing: `08-ai-model-layer.md#model-router`
- rag-search: `08-ai-model-layer.md#rag-semantic-search`
- context-summary: `05-context-engine.md#summary-rewrite-rules`
- crystallisation: `12-intent-framework.md#crystallisation`
- inactivity-detection: `11-feedback-loop.md#inactivity-detection`
- debrief: `11-feedback-loop.md#feedback-loop`
- source-adapters: `04-ingestion-pipeline.md#source-adapters-v1`
- pipeline-stages: `04-ingestion-pipeline.md#pipeline-stages`
- auth: `02-architecture.md#security`
- privacy: `01-vision.md#core-principles`
- infrastructure: `06-infrastructure.md#docker-services`, `06-infrastructure.md#dns-configuration`
- deployment: `06-infrastructure.md#deployment-workflow`
- slot-filling: `12-intent-framework.md#fill-strategies`, `12-intent-framework.md#slot-schema`

## Implementation Plans
- phase-3-email-ingestion: `plans/phase3-email-ingestion.md`
- phase-3b-clarification-queue: `plans/phase3b-clarification-queue.md`
- phase-4-frontend: `plans/phase4-frontend.md`
- phase-5-transaction-ingestion: `plans/phase5-transaction-ingestion.md`
- phase-4c-feature-completeness: `plans/phase4c-feature-completeness.md`
- phase-7a-daily-planner: `plans/phase7a-daily-planner.md`

## By Schema
- contexts: `03-data-model.md#contexts`
- context_events: removed in migration v1.25; context timelines now derive from `thread_entries` filtered on `subject_type='context'`
- tasks: `03-data-model.md#tasks`
- thread_entries: `03-data-model.md#thread-entries`
- events: `03-data-model.md#events`, `arch/event-backend.md`
- daily_plans: `03-data-model.md#daily-plans`, `arch/dailyplan-backend.md`
- daily_plan_items: `03-data-model.md#daily-plan-items`, `arch/dailyplan-backend.md`
- time_blocks: `03-data-model.md#time-blocks`
- task_dependencies: `03-data-model.md#task-dependencies`
- notes: `03-data-model.md#notes`, `arch/note-backend.md`
- note_tags: `03-data-model.md#notes`, `arch/note-backend.md`
- activity_logs: `03-data-model.md#activity-logs`, `arch/activitylog-backend.md`
- raw_inputs: `03-data-model.md#raw-inputs`
- emails: `03-data-model.md#emails`
- transactions: `03-data-model.md#transactions`, `arch/transaction-backend.md`
- tags: `03-data-model.md#tags`
- clarification_items: `03-data-model.md#clarification-items`, `10-clarification-patterns.md#clarification-items`
- inactivity_checks: `03-data-model.md#inactivity-checks`
- outcome_observations: `03-data-model.md#outcome-observations`, `11-feedback-loop.md#outcome-observations`
- pattern_observations: `10-clarification-patterns.md#pattern-observations` (future — Phase 5b)
- intent_adapters: `12-intent-framework.md#data-model`
- intent_executions: `12-intent-framework.md#data-model`
- workflow_creation_sessions: `12-intent-framework.md#data-model`
- embeddings: `03-data-model.md#embeddings`, `08-ai-model-layer.md#vector-storage-ddl`
- entity_links: `03-data-model.md#entity-links`
- transaction_splits: `03-data-model.md#transaction-splits`
- classification_corrections: `03-data-model.md#classification-corrections`
- task_dependencies: `03-data-model.md#task-dependencies`
