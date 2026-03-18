# Architecture

Single Go binary + Vue 3 frontend + SQLite, self-hosted via Docker Compose with nginx reverse proxy.

## Components
- **MCP server** (`POST /mcp`) — Claude's interface, JSON-RPC 2.0
- **REST API** (`/api/v1/...`) — frontend interface, standard JSON
- **Ingestion pipeline** — internal, processes raw data from source adapters
- **Intent framework** — agentic layer, recognises goals, fills slots, executes via adapters (see `12-intent-framework.md`)
- **AI model layer** — routes inference/embedding by sensitivity tier (see `08-ai-model-layer.md`)

## Source Adapters
Implement `Source` interface → feed ingestion pipeline via `RawInput`.

Built: (none yet — voice is indirect via Claude MCP)
Planned: SMTP receiver (email), CSV importer (transactions), receipt capture (photo)
Future: calendar (read-only), Apple Health

## Frontend
- Vue 3 + Vite, single codebase, two shells (web sidebar + mobile tab bar)
- Capacitor for native iOS
- See `09-frontend.md` for full component architecture

## Subdomain Routing
| Subdomain | Target |
|-----------|--------|
| app.domain.com | Vue frontend (port 5173/static) |
| api.domain.com | Go backend (port 8080) |
| mail.domain.com | SMTP receiver (port 25/587) |

## Request Flows

**Voice → task:** Siri shortcut → Claude API → reads SKILL.md → calls create_task via POST /mcp → Go validates + writes DB → Claude confirms

**Email → context update:** Forward email → SMTP stores raw → pipeline extracts → Claude identifies context + tasks + deadlines → writes entities → optional notification

**Transaction → context:** Upload CSV → parser normalises → pipeline processes → Claude matches to contexts → unmatched flagged for review

**Planning:** User asks Claude → list_tasks + list_contexts via MCP → reads calendar (when available) → reasons across data → proposes prioritised plan

## Security
- Single static API key (`X-API-Key` header or `Authorization: Bearer`)
- TLS via nginx, internal Docker traffic plain HTTP
- Rate limiting on `/mcp`
- No multi-tenancy, no user accounts
- Claude receives structured/summarised data, not raw PII

## Tech Choices

| Layer | Choice | Reason |
|-------|--------|--------|
| Backend | Go | Fast, single binary, low memory |
| Database | SQLite (WAL mode) | No service to manage, single-user sufficient |
| Vector search | sqlite-vec | Adds vectors to existing DB |
| MCP transport | Streamable HTTP | Claude connector standard |
| External inference | Anthropic API | Tier 1 + promoted Tier 2 |
| Local inference | Ollama | Tier 2 sanitization + Tier 3 |
| Local embeddings | nomic-embed-text via Ollama | 768-dim, fast, private |
| Frontend | Vue 3 + Vite | Fast dev, component-based |
| Mobile | Capacitor | Wraps Vue as native iOS |
| State | Pinia | Simple, Vue 3 integrated |
| Reverse proxy | nginx | TLS, rate limiting |
| Container | Docker Compose | Right complexity for personal app |
| ML service | Python + FastAPI | ML ecosystem, isolated |
