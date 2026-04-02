# TOC

Lookup index for `.docs/` planning files. Skills resolve `file#section` references by reading the target file and extracting content under the matching `##` heading through the next heading of equal or higher level.

## By Domain
- task: `03-data-model.md#tasks`, `07-roadmap.md#phase-1--working-core`
- context: `03-data-model.md#contexts`, `03-data-model.md#context-events`, `05-context-engine.md#context-operations`, `05-context-engine.md#context-lifecycle`, `07-roadmap.md#phase-2--contexts`, `arch/context-backend.md`
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
- timeblock: `03-data-model.md#time-blocks`, `07-roadmap.md#phase-7b--calendar-view--time-blocks`
- schedule: `07-roadmap.md#phase-7b--calendar-view--time-blocks`
- server: `06-infrastructure.md#monitoring`
- dailyplan: `03-data-model.md#daily-plans`, `03-data-model.md#daily-plan-items`, `05-context-engine.md#daily-plan-phase-7a`, `arch/dailyplan-backend.md`
- notes: `03-data-model.md#notes`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- recurring-tasks: `03-data-model.md#tasks`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- activity-logs: `03-data-model.md#activity-logs`, `07-roadmap.md#phase-7c--life-dashboard-primitives`
- frontend: `09-frontend.md#navigation-structure`, `09-frontend.md#shared-components`, `09-frontend.md#routes`, `09-frontend.md#pwa`, `07-roadmap.md#phase-4--frontend-web-shell`
- pwa: `09-frontend.md#pwa`, `07-roadmap.md#phase-4b--mobile-shell-capacitor`

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
- context_events: `03-data-model.md#context-events`
- tasks: `03-data-model.md#tasks`
- thread_entries: `03-data-model.md#thread-entries`
- events: `03-data-model.md#events`, `arch/event-backend.md`
- daily_plans: `03-data-model.md#daily-plans`, `arch/dailyplan-backend.md`
- daily_plan_items: `03-data-model.md#daily-plan-items`, `arch/dailyplan-backend.md`
- time_blocks: `03-data-model.md#time-blocks`
- notes: `03-data-model.md#notes` (future — Phase 7c)
- note_tags: `03-data-model.md#notes` (future — Phase 7c)
- activity_logs: `03-data-model.md#activity-logs` (future — Phase 7c)
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
- embeddings: `08-ai-model-layer.md#vector-storage-ddl`
