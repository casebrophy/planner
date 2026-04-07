# Auto-Classify and Link Entities Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Auto-classify any created note, task, or event into a context, and allow entities to be explicitly linked together (e.g., a wedding note linked to the wedding event), with AI suggestions surfaced via the clarification queue.

**Architecture:** Hybrid model — existing `context_id` FK remains the primary relationship hub; a new `entity_links` junction table adds direct cross-entity links (note→event, note→task, etc.). Classification is triggered async after every create (fire-and-forget goroutine in the app handler). Low-confidence AI matches surface as clarification cards; high-confidence (≥0.7) writes directly.

**Tech Stack:** Go (sqlx, uuid, json), PostgreSQL, Vue 3 + TypeScript (Pinia, Vitest)

---

## Scope Note

This plan has four independently shippable phases. Each phase produces working, testable software on its own:

- **Phase 1 (Tasks 1–6):** Manual entity linking via API (migration + entitylinkbus + entitylinkapp)
- **Phase 2 (Tasks 7–9):** AI classifies notes and events into contexts (extend classifyapp)
- **Phase 3 (Task 10):** Auto-trigger classification on every note/event create
- **Phase 4 (Tasks 11–14):** Frontend UI (types, service, store, Related Items panel, ClarificationCard)

Ship and test Phase 1 before starting Phase 2.

---

## File Map

| Action | File | Responsibility |
|--------|------|----------------|
| MODIFY | `business/sdk/migrate/sql/migrate.sql` | Add entity_links table (version 1.20) |
| CREATE | `business/domain/entitylinkbus/model.go` | EntityLink, NewEntityLink business types |
| CREATE | `business/domain/entitylinkbus/entitylinkbus.go` | Business + Storer interface |
| CREATE | `business/domain/entitylinkbus/stores/entitylinkdb/model.go` | DB struct + converters |
| CREATE | `business/domain/entitylinkbus/stores/entitylinkdb/entitylinkdb.go` | SQL CRUD |
| CREATE | `business/domain/entitylinkbus/entitylinkbus_test.go` | Store integration tests |
| CREATE | `app/domain/entitylinkapp/model.go` | App-layer DTOs + converters |
| CREATE | `app/domain/entitylinkapp/entitylinkapp.go` | HTTP handlers |
| CREATE | `app/domain/entitylinkapp/route.go` | Route registration |
| MODIFY | `business/sdk/dbtest/model.go` | Add EntityLink to BusDomain |
| MODIFY | `business/sdk/dbtest/business.go` | Wire entitylinkbus in newBusDomains |
| MODIFY | `api/services/planner/main.go` | Add entitylinkapp.Routes{} to WebAPI |
| MODIFY | `business/types/clarificationkind/clarificationkind.go` | Add EntityLink kind + weight |
| MODIFY | `business/domain/clarificationbus/options.go` | Add EntityLinkOptions struct |
| MODIFY | `app/domain/classifyapp/classifyapp.go` | Add entity_type routing + note/event classify |
| MODIFY | `app/domain/classifyapp/route.go` | Inject notebus, eventbus, entitylinkbus |
| MODIFY | `app/domain/noteapp/noteapp.go` | Fire async classify goroutine after create |
| MODIFY | `app/domain/noteapp/route.go` | Inject contextbus, clarificationbus, extractor |
| MODIFY | `app/domain/eventapp/eventapp.go` | Fire async classify goroutine after create |
| MODIFY | `app/domain/eventapp/route.go` | Inject contextbus, clarificationbus, extractor |
| CREATE | `web/src/types/entityLink.ts` | EntityLink, EntityKind, NewEntityLink TS types |
| MODIFY | `web/src/types/index.ts` | Re-export EntityLink types |
| CREATE | `web/src/services/entityLinkService.ts` | List + create + delete entity links |
| CREATE | `web/src/stores/entityLinkStore.ts` | Pinia store following CRUD factory pattern |
| MODIFY | `web/src/views/NoteDetailView.vue` | Add Related Items panel |
| MODIFY | `web/src/views/TaskDetailView.vue` | Add Related Items panel |
| MODIFY | `web/src/components/clarifications/ClarificationCard.vue` | Handle entity_link kind |

---

## Phase 1: Entity Link Plumbing

### Task 1: Migration — entity_links table

**Files:**
- Modify: `business/sdk/migrate/sql/migrate.sql`

- [ ] **Step 1: Append migration 1.20 to migrate.sql**

Add this block at the end of `business/sdk/migrate/sql/migrate.sql`:

```sql
-- Version: 1.20
-- Description: Create entity_links table for cross-entity semantic linking
CREATE TABLE entity_links (
    link_id      UUID        NOT NULL DEFAULT gen_random_uuid(),
    source_type  TEXT        NOT NULL CHECK (source_type IN ('task', 'note', 'event')),
    source_id    UUID        NOT NULL,
    target_type  TEXT        NOT NULL CHECK (target_type IN ('task', 'note', 'event')),
    target_id    UUID        NOT NULL,
    confidence   FLOAT8      NOT NULL DEFAULT 1.0,
    kind         TEXT        NOT NULL DEFAULT 'manual' CHECK (kind IN ('manual', 'ai_suggested')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (link_id),
    CONSTRAINT entity_links_no_self_link CHECK (
        NOT (source_type = target_type AND source_id = target_id)
    )
);
CREATE UNIQUE INDEX idx_entity_links_pair   ON entity_links(source_type, source_id, target_type, target_id);
CREATE INDEX        idx_entity_links_source ON entity_links(source_type, source_id);
CREATE INDEX        idx_entity_links_target ON entity_links(target_type, target_id);
```

- [ ] **Step 2: Run migration**

```bash
make migrate
```

Expected: `migration 1.20 applied` (no errors)

- [ ] **Step 3: Commit**

```bash
git add business/sdk/migrate/sql/migrate.sql
git commit -m "feat: add entity_links table (migration 1.20)"
```

---

### Task 2: entitylinkbus model + Storer interface

**Files:**
- Create: `business/domain/entitylinkbus/model.go`
- Create: `business/domain/entitylinkbus/entitylinkbus.go`

- [ ] **Step 1: Create `business/domain/entitylinkbus/model.go`**

```go
package entitylinkbus

import (
	"time"

	"github.com/google/uuid"
)

// EntityLink is a directional semantic link between two entities.
// At query time, always fetch both directions (QueryBySource + QueryByTarget).
type EntityLink struct {
	ID         uuid.UUID
	SourceType string // "task" | "note" | "event"
	SourceID   uuid.UUID
	TargetType string // "task" | "note" | "event"
	TargetID   uuid.UUID
	Confidence float64 // 1.0 for manual; 0.0–1.0 for AI-suggested
	Kind       string  // "manual" | "ai_suggested"
	CreatedAt  time.Time
}

// NewEntityLink is the input for creating an entity link.
type NewEntityLink struct {
	SourceType string
	SourceID   uuid.UUID
	TargetType string
	TargetID   uuid.UUID
	Confidence float64
	Kind       string
}
```

- [ ] **Step 2: Create `business/domain/entitylinkbus/entitylinkbus.go`**

```go
package entitylinkbus

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/foundation/logger"
)

// Storer defines the persistence interface for entity links.
type Storer interface {
	Create(ctx context.Context, link EntityLink) error
	Delete(ctx context.Context, id uuid.UUID) error
	QueryBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]EntityLink, error)
	QueryByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]EntityLink, error)
}

// Business manages entity link operations.
type Business struct {
	log    *logger.Logger
	storer Storer
}

// NewBusiness constructs an entitylinkbus.Business.
func NewBusiness(log *logger.Logger, storer Storer) *Business {
	return &Business{log: log, storer: storer}
}

// Create persists a new entity link.
func (b *Business) Create(ctx context.Context, nl NewEntityLink) (EntityLink, error) {
	if nl.Kind == "" {
		nl.Kind = "manual"
	}
	if nl.Confidence == 0 && nl.Kind == "manual" {
		nl.Confidence = 1.0
	}

	link := EntityLink{
		ID:         uuid.New(),
		SourceType: nl.SourceType,
		SourceID:   nl.SourceID,
		TargetType: nl.TargetType,
		TargetID:   nl.TargetID,
		Confidence: nl.Confidence,
		Kind:       nl.Kind,
		CreatedAt:  time.Now(),
	}

	if err := b.storer.Create(ctx, link); err != nil {
		return EntityLink{}, fmt.Errorf("create: %w", err)
	}

	return link, nil
}

// Delete removes an entity link by ID.
func (b *Business) Delete(ctx context.Context, id uuid.UUID) error {
	if err := b.storer.Delete(ctx, id); err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	return nil
}

// QueryByEntity returns all links where the entity appears as source or target.
func (b *Business) QueryByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]EntityLink, error) {
	bySource, err := b.storer.QueryBySource(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("query by source: %w", err)
	}

	byTarget, err := b.storer.QueryByTarget(ctx, entityType, entityID)
	if err != nil {
		return nil, fmt.Errorf("query by target: %w", err)
	}

	return append(bySource, byTarget...), nil
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./business/domain/entitylinkbus/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add business/domain/entitylinkbus/
git commit -m "feat: add entitylinkbus model and Storer interface"
```

---

### Task 3: entitylinkdb store implementation

**Files:**
- Create: `business/domain/entitylinkbus/stores/entitylinkdb/model.go`
- Create: `business/domain/entitylinkbus/stores/entitylinkdb/entitylinkdb.go`

- [ ] **Step 1: Create `business/domain/entitylinkbus/stores/entitylinkdb/model.go`**

```go
package entitylinkdb

import (
	"time"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
)

type entityLinkDB struct {
	ID         uuid.UUID `db:"link_id"`
	SourceType string    `db:"source_type"`
	SourceID   uuid.UUID `db:"source_id"`
	TargetType string    `db:"target_type"`
	TargetID   uuid.UUID `db:"target_id"`
	Confidence float64   `db:"confidence"`
	Kind       string    `db:"kind"`
	CreatedAt  time.Time `db:"created_at"`
}

func toDBEntityLink(l entitylinkbus.EntityLink) entityLinkDB {
	return entityLinkDB{
		ID:         l.ID,
		SourceType: l.SourceType,
		SourceID:   l.SourceID,
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt,
	}
}

func toBusEntityLink(l entityLinkDB) entitylinkbus.EntityLink {
	return entitylinkbus.EntityLink{
		ID:         l.ID,
		SourceType: l.SourceType,
		SourceID:   l.SourceID,
		TargetType: l.TargetType,
		TargetID:   l.TargetID,
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt,
	}
}
```

- [ ] **Step 2: Create `business/domain/entitylinkbus/stores/entitylinkdb/entitylinkdb.go`**

```go
package entitylinkdb

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/foundation/logger"
)

// Store implements entitylinkbus.Storer against PostgreSQL.
type Store struct {
	log *logger.Logger
	db  sqlx.ExtContext
}

// NewStore constructs an entitylinkdb.Store.
func NewStore(log *logger.Logger, db *sqlx.DB) *Store {
	return &Store{log: log, db: db}
}

func (s *Store) Create(ctx context.Context, link entitylinkbus.EntityLink) error {
	const q = `
	INSERT INTO entity_links (link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at)
	VALUES (:link_id, :source_type, :source_id, :target_type, :target_id, :confidence, :kind, :created_at)`

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, toDBEntityLink(link)); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}

func (s *Store) Delete(ctx context.Context, id uuid.UUID) error {
	const q = `DELETE FROM entity_links WHERE link_id = :link_id`

	data := struct {
		ID uuid.UUID `db:"link_id"`
	}{ID: id}

	if err := sqldb.NamedExecContext(ctx, s.log, s.db, q, data); err != nil {
		return fmt.Errorf("namedexeccontext: %w", err)
	}
	return nil
}

func (s *Store) QueryBySource(ctx context.Context, sourceType string, sourceID uuid.UUID) ([]entitylinkbus.EntityLink, error) {
	const q = `
	SELECT link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at
	FROM entity_links
	WHERE source_type = :source_type AND source_id = :source_id
	ORDER BY created_at DESC`

	data := map[string]any{
		"source_type": sourceType,
		"source_id":   sourceID,
	}

	rows, err := sqldb.NamedQuerySlice[entityLinkDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	links := make([]entitylinkbus.EntityLink, len(rows))
	for i, r := range rows {
		links[i] = toBusEntityLink(r)
	}
	return links, nil
}

func (s *Store) QueryByTarget(ctx context.Context, targetType string, targetID uuid.UUID) ([]entitylinkbus.EntityLink, error) {
	const q = `
	SELECT link_id, source_type, source_id, target_type, target_id, confidence, kind, created_at
	FROM entity_links
	WHERE target_type = :target_type AND target_id = :target_id
	ORDER BY created_at DESC`

	data := map[string]any{
		"target_type": targetType,
		"target_id":   targetID,
	}

	rows, err := sqldb.NamedQuerySlice[entityLinkDB](ctx, s.log, s.db, q, data)
	if err != nil {
		return nil, fmt.Errorf("namedqueryslice: %w", err)
	}

	links := make([]entitylinkbus.EntityLink, len(rows))
	for i, r := range rows {
		links[i] = toBusEntityLink(r)
	}
	return links, nil
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./business/domain/entitylinkbus/...
```

Expected: no errors

- [ ] **Step 4: Write store integration test**

Create `business/domain/entitylinkbus/entitylinkbus_test.go`:

```go
package entitylinkbus_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/business/sdk/dbtest"
)

func Test_EntityLink(t *testing.T) {
	t.Parallel()

	db := dbtest.New(t, "Test_EntityLink")
	store := entitylinkdb.NewStore(db.Log, db.DB)
	bus := entitylinkbus.NewBusiness(db.Log, store)
	ctx := context.Background()

	// Seed two tasks via dbtest so we have valid UUIDs to link
	tasks, err := db.BusDomain.Task.Query(ctx, db.BusDomain.Task.NewQueryFilter(), db.BusDomain.Task.DefaultOrderBy(), dbtest.Page1)
	// Tasks may be empty; we just need two UUIDs for testing links
	_ = tasks
	_ = err

	// Use fixed UUIDs — entity_links has no FK constraints to tasks/notes/events
	sourceID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	targetID := uuid.MustParse("22222222-2222-2222-2222-222222222222")

	t.Run("create_and_query_by_source", func(t *testing.T) {
		link, err := bus.Create(ctx, entitylinkbus.NewEntityLink{
			SourceType: "note",
			SourceID:   sourceID,
			TargetType: "event",
			TargetID:   targetID,
			Kind:       "manual",
		})
		if err != nil {
			t.Fatalf("Create: %s", err)
		}

		links, err := bus.QueryByEntity(ctx, "note", sourceID)
		if err != nil {
			t.Fatalf("QueryByEntity: %s", err)
		}

		if len(links) != 1 {
			t.Fatalf("expected 1 link, got %d", len(links))
		}

		diff := cmp.Diff(link, links[0], cmpopts.EquateApproxTime(time.Second))
		if diff != "" {
			t.Errorf("link mismatch:\n%s", diff)
		}
	})

	t.Run("query_by_target_sees_same_link", func(t *testing.T) {
		links, err := bus.QueryByEntity(ctx, "event", targetID)
		if err != nil {
			t.Fatalf("QueryByEntity: %s", err)
		}
		if len(links) == 0 {
			t.Fatal("expected at least 1 link from target side")
		}
	})

	t.Run("delete", func(t *testing.T) {
		link, err := bus.Create(ctx, entitylinkbus.NewEntityLink{
			SourceType: "task",
			SourceID:   sourceID,
			TargetType: "note",
			TargetID:   targetID,
			Kind:       "manual",
		})
		if err != nil {
			t.Fatalf("Create for delete: %s", err)
		}

		if err := bus.Delete(ctx, link.ID); err != nil {
			t.Fatalf("Delete: %s", err)
		}

		remaining, err := bus.QueryByEntity(ctx, "task", sourceID)
		if err != nil {
			t.Fatalf("QueryByEntity after delete: %s", err)
		}
		for _, l := range remaining {
			if l.ID == link.ID {
				t.Error("deleted link still present")
			}
		}
	})
}
```

> **Note:** The test uses fixed UUIDs because `entity_links` has no FK constraints to entity tables (by design — links can outlive reordering, and tests don't need real tasks). Import `github.com/google/uuid` in the test file.

- [ ] **Step 5: Run test**

```bash
go test ./business/domain/entitylinkbus/... -run Test_EntityLink -count=1 -v
```

Expected: PASS (3 sub-tests)

- [ ] **Step 6: Commit**

```bash
git add business/domain/entitylinkbus/
git commit -m "feat: add entitylinkdb store + integration tests"
```

---

### Task 4: entitylinkapp HTTP layer

**Files:**
- Create: `app/domain/entitylinkapp/model.go`
- Create: `app/domain/entitylinkapp/entitylinkapp.go`
- Create: `app/domain/entitylinkapp/route.go`

- [ ] **Step 1: Create `app/domain/entitylinkapp/model.go`**

```go
package entitylinkapp

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/business/domain/entitylinkbus"
)

// EntityLink is the app-layer representation of an entity link.
type EntityLink struct {
	ID         string  `json:"id"`
	SourceType string  `json:"sourceType"`
	SourceID   string  `json:"sourceId"`
	TargetType string  `json:"targetType"`
	TargetID   string  `json:"targetId"`
	Confidence float64 `json:"confidence"`
	Kind       string  `json:"kind"`
	CreatedAt  string  `json:"createdAt"`
}

func (e EntityLink) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

// EntityLinks is a slice of EntityLink with an Encode method.
type EntityLinks struct {
	Items []EntityLink `json:"items"`
	Total int          `json:"total"`
}

func (e EntityLinks) Encode() ([]byte, string, error) {
	data, err := json.Marshal(e)
	return data, "application/json", err
}

// NewEntityLink is the request body for creating a link.
type NewEntityLink struct {
	SourceType string `json:"sourceType"`
	SourceID   string `json:"sourceId"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
}

func toAppEntityLink(l entitylinkbus.EntityLink) EntityLink {
	return EntityLink{
		ID:         l.ID.String(),
		SourceType: l.SourceType,
		SourceID:   l.SourceID.String(),
		TargetType: l.TargetType,
		TargetID:   l.TargetID.String(),
		Confidence: l.Confidence,
		Kind:       l.Kind,
		CreatedAt:  l.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func toBusNewEntityLink(n NewEntityLink) (entitylinkbus.NewEntityLink, error) {
	sourceID, err := uuid.Parse(n.SourceID)
	if err != nil {
		return entitylinkbus.NewEntityLink{}, fmt.Errorf("sourceId: %w", err)
	}

	targetID, err := uuid.Parse(n.TargetID)
	if err != nil {
		return entitylinkbus.NewEntityLink{}, fmt.Errorf("targetId: %w", err)
	}

	return entitylinkbus.NewEntityLink{
		SourceType: n.SourceType,
		SourceID:   sourceID,
		TargetType: n.TargetType,
		TargetID:   targetID,
		Kind:       "manual",
		Confidence: 1.0,
	}, nil
}
```

- [ ] **Step 2: Create `app/domain/entitylinkapp/entitylinkapp.go`**

```go
package entitylinkapp

import (
	"context"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	entityLinkBus *entitylinkbus.Business
}

// queryByEntity returns all links (source or target) for a given entity.
// Query params: entity_type (task|note|event), entity_id (UUID)
func (a *app) queryByEntity(ctx context.Context, r *http.Request) web.Encoder {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		return errs.Newf(errs.InvalidArgument, "entity_type is required")
	}

	entityIDStr := r.URL.Query().Get("entity_id")
	entityID, err := uuid.Parse(entityIDStr)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	links, err := a.entityLinkBus.QueryByEntity(ctx, entityType, entityID)
	if err != nil {
		return errs.Newf(errs.Internal, "query: %s", err)
	}

	items := make([]EntityLink, len(links))
	for i, l := range links {
		items[i] = toAppEntityLink(l)
	}

	return EntityLinks{Items: items, Total: len(items)}
}

// create creates a new manual entity link.
func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewEntityLink
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	busNew, err := toBusNewEntityLink(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	link, err := a.entityLinkBus.Create(ctx, busNew)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	return toAppEntityLink(link)
}

// delete removes an entity link by ID.
func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "link_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if err := a.entityLinkBus.Delete(ctx, id); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}

	return query.Result[struct{}]{} // 204-style: empty JSON object
}
```

> **Note:** `query.Result[struct{}]{}` is a placeholder for the empty-response pattern. Check how `noteapp.go` handles delete — if it returns `nil`, use the same pattern. If the codebase uses a specific empty-response type, use that instead.

- [ ] **Step 3: Create `app/domain/entitylinkapp/route.go`**

```go
package entitylinkapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes registers entitylinkapp routes.
type Routes struct{}

// Add wires up the entity link endpoints.
func (Routes) Add(a *web.App, cfg mux.Config) {
	store := entitylinkdb.NewStore(cfg.Log, cfg.DB)
	bus := entitylinkbus.NewBusiness(cfg.Log, store)

	hdl := &app{entityLinkBus: bus}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/entity-links", hdl.queryByEntity, authen)
	a.Handle(http.MethodPost, "/api/v1/entity-links", hdl.create, authen)
	a.Handle(http.MethodDelete, "/api/v1/entity-links/{link_id}", hdl.delete, authen)
}
```

- [ ] **Step 4: Fix the delete return value**

Check how `noteapp.go` handles `delete`:

```bash
grep -A5 "func (a \*app) delete" /Users/casebrophy/personal/planner/app/domain/noteapp/noteapp.go
```

If it returns `nil`, update `entitylinkapp.go` to also return `nil`:

```go
func (a *app) delete(ctx context.Context, r *http.Request) web.Encoder {
	id, err := uuid.Parse(web.Param(r, "link_id"))
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}
	if err := a.entityLinkBus.Delete(ctx, id); err != nil {
		return errs.Newf(errs.Internal, "delete: %s", err)
	}
	return nil
}
```

- [ ] **Step 5: Compile check**

```bash
go build ./app/domain/entitylinkapp/...
```

Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add app/domain/entitylinkapp/
git commit -m "feat: add entitylinkapp HTTP handlers and routes"
```

---

### Task 5: Wire entitylinkbus into dbtest and main.go

**Files:**
- Modify: `business/sdk/dbtest/model.go`
- Modify: `business/sdk/dbtest/business.go`
- Modify: `api/services/planner/main.go`

- [ ] **Step 1: Add EntityLink field to `business/sdk/dbtest/model.go`**

In `model.go`, add the import and field:

```go
import (
	// ... existing imports ...
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
)

type BusDomain struct {
	// ... existing fields ...
	EntityLink    *entitylinkbus.Business
}
```

- [ ] **Step 2: Wire entitylinkbus in `business/sdk/dbtest/business.go`**

Add to `newBusDomains`:

```go
import (
	// ... existing imports ...
	"github.com/casebrophy/planner/business/domain/entitylinkbus"
	"github.com/casebrophy/planner/business/domain/entitylinkbus/stores/entitylinkdb"
)

func newBusDomains(log *logger.Logger, db *sqlx.DB) BusDomain {
	// ... existing wiring ...
	entityLinkBus := entitylinkbus.NewBusiness(log, entitylinkdb.NewStore(log, db))

	return BusDomain{
		// ... existing fields ...
		EntityLink: entityLinkBus,
	}
}
```

- [ ] **Step 3: Add entitylinkapp.Routes{} to `api/services/planner/main.go`**

Add the import:
```go
"github.com/casebrophy/planner/app/domain/entitylinkapp"
```

Add to the `mux.WebAPI` call (after `classifyapp.Routes{}`):
```go
handler := mux.WebAPI(muxCfg,
    // ... existing routes ...
    classifyapp.Routes{},
    entitylinkapp.Routes{},
)
```

- [ ] **Step 4: Build the full service**

```bash
go build ./api/services/planner/...
```

Expected: no errors

- [ ] **Step 5: Run all tests**

```bash
make test
```

Expected: all pass

- [ ] **Step 6: Commit**

```bash
git add business/sdk/dbtest/ api/services/planner/main.go
git commit -m "feat: wire entitylinkbus into dbtest and main"
```

---

### Task 6: Smoke-test Phase 1 end-to-end

- [ ] **Step 1: Start the dev stack**

```bash
make dev-up
```

- [ ] **Step 2: Create a manual entity link**

```bash
curl -s -X POST http://localhost:8080/api/v1/entity-links \
  -H "X-API-Key: devkey123" \
  -H "Content-Type: application/json" \
  -d '{
    "sourceType": "note",
    "sourceId": "00000000-0000-0000-0000-000000000001",
    "targetType": "event",
    "targetId": "00000000-0000-0000-0000-000000000002"
  }' | jq .
```

Expected: `{"id":"...","sourceType":"note","sourceId":"...","targetType":"event","targetId":"...","confidence":1,"kind":"manual","createdAt":"..."}`

> The UUIDs don't need to exist — no FK constraints on entity_links.

- [ ] **Step 3: Query the link back**

```bash
LINK_ID=$(curl -s -X POST http://localhost:8080/api/v1/entity-links \
  -H "X-API-Key: devkey123" \
  -H "Content-Type: application/json" \
  -d '{"sourceType":"task","sourceId":"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa","targetType":"note","targetId":"bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"}' | jq -r .id)

curl -s "http://localhost:8080/api/v1/entity-links?entity_type=task&entity_id=aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa" \
  -H "X-API-Key: devkey123" | jq .
```

Expected: `{"items":[{...}],"total":1}`

- [ ] **Step 4: Stop dev stack**

```bash
make dev-down
```

- [ ] **Step 5: Commit (if any fixes needed)**

```bash
git add -A && git commit -m "fix: entity link smoke test fixes"
```

---

## Phase 2: Classification Extension

### Task 7: Add EntityLink clarification kind

**Files:**
- Modify: `business/types/clarificationkind/clarificationkind.go`
- Modify: `business/domain/clarificationbus/options.go`

- [ ] **Step 1: Add `EntityLink` kind to `clarificationkind.go`**

In `clarificationkind.go`, add after `TaskDebrief`:

```go
var (
	// ... existing kinds ...
	EntityLink = Kind{"entity_link"}
)

var kinds = map[string]Kind{
	// ... existing entries ...
	EntityLink.value: EntityLink,
}

var AllKinds = []Kind{
	// ... existing ...
	EntityLink,
}

var KindWeights = map[Kind]float32{
	// ... existing ...
	EntityLink: 0.7,
}
```

- [ ] **Step 2: Add `EntityLinkOptions` to `business/domain/clarificationbus/options.go`**

Append to `options.go`:

```go
// EntityLinkOptions is the typed answer options for entity_link clarifications.
// Describes a suggested link between two entities.
type EntityLinkOptions struct {
	SourceType string  `json:"source_type"`
	SourceID   string  `json:"source_id"`
	TargetType string  `json:"target_type"`
	TargetID   string  `json:"target_id"`
	Confidence float64 `json:"confidence"`
}
```

- [ ] **Step 3: Compile check**

```bash
go build ./business/types/clarificationkind/... ./business/domain/clarificationbus/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add business/types/clarificationkind/clarificationkind.go \
        business/domain/clarificationbus/options.go
git commit -m "feat: add entity_link clarification kind and options"
```

---

### Task 8: Extend classifyapp to classify notes and events

**Files:**
- Modify: `app/domain/classifyapp/classifyapp.go`
- Modify: `app/domain/classifyapp/route.go`

- [ ] **Step 1: Rewrite `app/domain/classifyapp/classifyapp.go`**

Replace the existing file content with:

```go
package classifyapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/business/types/taskstatus"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	taskBus          *taskbus.Business
	noteBus          *notebus.Business
	eventBus         *eventbus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	extractor        extractor.Extractor
}

// classify is the single classify endpoint — entity_type query param selects which entity type to classify.
// Default is "task" for backward compatibility.
// POST /api/v1/classify?entity_type=task|note|event
func (a *app) classify(ctx context.Context, r *http.Request) web.Encoder {
	entityType := r.URL.Query().Get("entity_type")
	if entityType == "" {
		entityType = "task"
	}

	switch entityType {
	case "task":
		return a.classifyTasks(ctx)
	case "note":
		return a.classifyNotes(ctx)
	case "event":
		return a.classifyEvents(ctx)
	default:
		return errs.Newf(errs.InvalidArgument, "unsupported entity_type %q (use task, note, or event)", entityType)
	}
}

func (a *app) classifyTasks(ctx context.Context) web.Encoder {
	openStatus := taskstatus.Open
	tasks, err := a.taskBus.Query(ctx, taskbus.QueryFilter{Status: &openStatus}, taskbus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query tasks: %s", err)
	}

	var unlinked []taskbus.Task
	for _, t := range tasks {
		if t.ContextID == nil {
			unlinked = append(unlinked, t)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked tasks to classify", UnlinkedCount: 0}
	}

	ctxRefs, err := a.fetchContextRefs(ctx)
	if err != nil {
		return err
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, task := range unlinked {
			a.classifyEntity(bgCtx, "task", task.ID, fmt.Sprintf("Task: %s\nDescription: %s", task.Title, task.Description), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d tasks", n), UnlinkedCount: n}
}

func (a *app) classifyNotes(ctx context.Context) web.Encoder {
	notes, err := a.noteBus.Query(ctx, notebus.QueryFilter{}, notebus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query notes: %s", err)
	}

	var unlinked []notebus.Note
	for _, n := range notes {
		if n.ContextID == nil {
			unlinked = append(unlinked, n)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked notes to classify", UnlinkedCount: 0}
	}

	ctxRefs, errEnc := a.fetchContextRefs(ctx)
	if errEnc != nil {
		return errEnc
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, note := range unlinked {
			a.classifyEntity(bgCtx, "note", note.ID, fmt.Sprintf("Note: %s", note.Content), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d notes", n), UnlinkedCount: n}
}

func (a *app) classifyEvents(ctx context.Context) web.Encoder {
	events, err := a.eventBus.Query(ctx, eventbus.QueryFilter{}, eventbus.DefaultOrderBy, page.New(1, 200))
	if err != nil {
		return errs.Newf(errs.Internal, "query events: %s", err)
	}

	var unlinked []eventbus.Event
	for _, e := range events {
		if e.ContextID == nil {
			unlinked = append(unlinked, e)
		}
	}

	if len(unlinked) == 0 {
		return ClassifyAccepted{Message: "No unlinked events to classify", UnlinkedCount: 0}
	}

	ctxRefs, errEnc := a.fetchContextRefs(ctx)
	if errEnc != nil {
		return errEnc
	}

	n := len(unlinked)
	go func() {
		bgCtx := context.Background()
		for _, event := range unlinked {
			a.classifyEntity(bgCtx, "event", event.ID, fmt.Sprintf("Event: %s\nDescription: %s", event.Title, event.Description), ctxRefs)
		}
	}()

	return ClassifyAccepted{Message: fmt.Sprintf("Classification started for %d events", n), UnlinkedCount: n}
}

// fetchContextRefs builds the active context list for the extractor.
func (a *app) fetchContextRefs(ctx context.Context) ([]extractor.ContextRef, web.Encoder) {
	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return nil, errs.Newf(errs.Internal, "query contexts: %s", err)
	}

	refs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		refs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}
	return refs, nil
}

// classifyEntity extracts a context suggestion for a single entity and either updates it directly
// (confidence ≥ 0.7) or creates a clarification card (confidence < 0.7).
// Must be called in a background goroutine — uses context.Background().
func (a *app) classifyEntity(ctx context.Context, entityType string, entityID uuid.UUID, text string, ctxRefs []extractor.ContextRef) {
	extraction, err := a.extractor.ExtractText(ctx, text, ctxRefs)
	if err != nil {
		return
	}

	if extraction.SuggestedContextID == nil || *extraction.SuggestedContextID == "" {
		return
	}

	ctxID, err := uuid.Parse(*extraction.SuggestedContextID)
	if err != nil {
		return
	}

	if _, err := a.contextBus.QueryByID(ctx, ctxID); err != nil {
		return
	}

	if extraction.ContextConfidence >= 0.7 {
		switch entityType {
		case "task":
			ut := taskbus.UpdateTask{ContextID: &ctxID}
			task, err := a.taskBus.QueryByID(ctx, entityID)
			if err != nil {
				return
			}
			a.taskBus.Update(ctx, task, ut) //nolint:errcheck
		case "note":
			un := notebus.UpdateNote{ContextID: &ctxID}
			note, err := a.noteBus.QueryByID(ctx, entityID)
			if err != nil {
				return
			}
			a.noteBus.Update(ctx, note, un) //nolint:errcheck
		case "event":
			ue := eventbus.UpdateEvent{ContextID: &ctxID}
			event, err := a.eventBus.QueryByID(ctx, entityID)
			if err != nil {
				return
			}
			a.eventBus.Update(ctx, event, ue) //nolint:errcheck
		}
	} else {
		busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
		for i, r := range ctxRefs {
			busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
		}
		optJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
			SuggestedContext:  ctxID.String(),
			Confidence:        extraction.ContextConfidence,
			AvailableContexts: busCtxRefs,
		})
		guess, _ := json.Marshal(map[string]string{"context_id": ctxID.String()})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched %s to context with %.0f%% confidence", entityType, extraction.ContextConfidence*100)

		a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{ //nolint:errcheck
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   entityType,
			SubjectID:     entityID,
			Question:      fmt.Sprintf("Which context does this %s belong to?", entityType),
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optJSON),
		})
	}
}
```

- [ ] **Step 2: Update `app/domain/classifyapp/route.go` to inject notebus and eventbus**

Replace the existing route.go content:

```go
package classifyapp

import (
	"net/http"

	"github.com/casebrophy/planner/app/sdk/mid"
	"github.com/casebrophy/planner/app/sdk/mux"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
	"github.com/casebrophy/planner/business/domain/eventbus"
	"github.com/casebrophy/planner/business/domain/eventbus/stores/eventdb"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/domain/notebus/stores/notedb"
	"github.com/casebrophy/planner/business/domain/taskbus"
	"github.com/casebrophy/planner/business/domain/taskbus/stores/taskdb"
	"github.com/casebrophy/planner/foundation/web"
)

// Routes registers classifyapp routes.
type Routes struct{}

// Add wires up the classify endpoint.
func (Routes) Add(a *web.App, cfg mux.Config) {
	taskStore := taskdb.NewStore(cfg.Log, cfg.DB)
	depStore := taskdb.NewDependencyStore(cfg.Log, cfg.DB)
	taskBus := taskbus.NewBusiness(cfg.Log, taskStore, depStore)

	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	evtStore := eventdb.NewStore(cfg.Log, cfg.DB)
	evtBus := eventbus.NewBusiness(cfg.Log, evtStore)

	ctxStore := contextdb.NewStore(cfg.Log, cfg.DB)
	ctxBus := contextbus.NewBusiness(cfg.Log, ctxStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	claudeExt := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
	var ext extractor.Extractor = claudeExt
	if cfg.OllamaEnabled && cfg.OllamaURL != "" {
		ollamaExt := extractor.NewOllamaExtractor(cfg.OllamaURL, cfg.OllamaModel)
		ext = extractor.NewFailoverExtractor(cfg.Log, claudeExt, ollamaExt)
	}

	hdl := &app{
		taskBus:          taskBus,
		noteBus:          noteBus,
		eventBus:         evtBus,
		contextBus:       ctxBus,
		clarificationBus: clBus,
		extractor:        ext,
	}

	authen := mid.Auth(cfg.APIKey)

	// Existing route kept for backward compat (default entity_type=task)
	a.Handle(http.MethodPost, "/api/v1/tasks/classify", hdl.classify, authen)
	// New unified endpoint
	a.Handle(http.MethodPost, "/api/v1/classify", hdl.classify, authen)
}
```

- [ ] **Step 3: Build and test**

```bash
go build ./app/domain/classifyapp/...
make test
```

Expected: builds cleanly, all tests pass

- [ ] **Step 4: Commit**

```bash
git add app/domain/classifyapp/
git commit -m "feat: extend classifyapp to classify notes and events via entity_type param"
```

---

### Task 9: Implement clarification resolution side-effects for context_assignment

**Files:**
- Modify: `app/domain/clarificationapp/clarificationapp.go` (find the resolve handler)

The clarification resolution handler currently has a TODO for side-effects. This task implements them for `context_assignment` (which now applies to tasks, notes, and events).

- [ ] **Step 1: Find the resolve handler**

```bash
grep -n "Resolve\|resolve\|side.effect\|TODO" \
  /Users/casebrophy/personal/planner/app/domain/clarificationapp/clarificationapp.go | head -20
```

- [ ] **Step 2: Add context_assignment resolution side-effect**

In the resolve handler, after `clarificationBus.Resolve()` succeeds, add:

```go
// Apply resolution side-effects
if item.Kind == clarificationkind.ContextAssignment {
    var answer struct {
        ContextID string `json:"context_id"`
    }
    if err := json.Unmarshal([]byte(rc.Answer), &answer); err == nil && answer.ContextID != "" {
        ctxID, err := uuid.Parse(answer.ContextID)
        if err == nil {
            switch item.SubjectType {
            case "task":
                task, err := a.taskBus.QueryByID(ctx, item.SubjectID)
                if err == nil {
                    a.taskBus.Update(ctx, task, taskbus.UpdateTask{ContextID: &ctxID}) //nolint:errcheck
                }
            case "note":
                note, err := a.noteBus.QueryByID(ctx, item.SubjectID)
                if err == nil {
                    a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &ctxID}) //nolint:errcheck
                }
            case "event":
                event, err := a.eventBus.QueryByID(ctx, item.SubjectID)
                if err == nil {
                    a.eventBus.Update(ctx, event, eventbus.UpdateEvent{ContextID: &ctxID}) //nolint:errcheck
                }
            }
        }
    }
}
```

> You will need to inject `taskBus`, `noteBus`, `eventBus` into `clarificationapp.app` and update `clarificationapp/route.go` accordingly. Follow the same dependency-injection pattern as `classifyapp/route.go`.

- [ ] **Step 3: Build**

```bash
go build ./app/domain/clarificationapp/...
```

Expected: no errors

- [ ] **Step 4: Commit**

```bash
git add app/domain/clarificationapp/
git commit -m "feat: implement context_assignment clarification resolution side-effects"
```

---

## Phase 3: Async Post-Create Classification

### Task 10: Fire classify goroutine after note and event create

**Files:**
- Modify: `app/domain/noteapp/noteapp.go`
- Modify: `app/domain/noteapp/route.go`
- Modify: `app/domain/eventapp/eventapp.go`
- Modify: `app/domain/eventapp/route.go`

The classify goroutine belongs in the **app handler** (not the bus). This keeps the business layer dependency-free.

- [ ] **Step 1: Add classifier dependencies to `app/domain/noteapp/noteapp.go`**

Replace the `app` struct and `create` method in `noteapp.go`:

```go
package noteapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"

	"github.com/casebrophy/planner/app/sdk/errs"
	"github.com/casebrophy/planner/app/sdk/query"
	"github.com/casebrophy/planner/business/domain/clarificationbus"
	"github.com/casebrophy/planner/business/domain/contextbus"
	"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
	"github.com/casebrophy/planner/business/domain/notebus"
	"github.com/casebrophy/planner/business/sdk/page"
	"github.com/casebrophy/planner/business/sdk/sqldb"
	"github.com/casebrophy/planner/business/types/clarificationkind"
	"github.com/casebrophy/planner/foundation/web"
)

type app struct {
	noteBus          *notebus.Business
	contextBus       *contextbus.Business
	clarificationBus *clarificationbus.Business
	extractor        extractor.Extractor
}

func (a *app) create(ctx context.Context, r *http.Request) web.Encoder {
	var input NewNote
	if err := web.Decode(r, &input); err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	if input.Content == "" {
		return errs.Newf(errs.InvalidArgument, "content is required")
	}

	bnn, err := toBusNewNote(input)
	if err != nil {
		return errs.New(errs.InvalidArgument, err)
	}

	note, err := a.noteBus.Create(ctx, bnn)
	if err != nil {
		return errs.Newf(errs.Internal, "create: %s", err)
	}

	// Fire async classify only if note has no context yet
	if note.ContextID == nil {
		go a.asyncClassify(context.Background(), "note", note.ID, fmt.Sprintf("Note: %s", note.Content))
	}

	return toAppNote(note)
}

// asyncClassify runs in a background goroutine — never blocks the HTTP response.
func (a *app) asyncClassify(ctx context.Context, entityType string, entityID uuid.UUID, text string) {
	activeStatus := contextbus.Active
	contexts, err := a.contextBus.Query(ctx, contextbus.QueryFilter{Status: &activeStatus}, contextbus.DefaultOrderBy, page.New(1, 50))
	if err != nil {
		return
	}

	ctxRefs := make([]extractor.ContextRef, len(contexts))
	for i, c := range contexts {
		ctxRefs[i] = extractor.ContextRef{ID: c.ID.String(), Title: c.Title}
	}

	extraction, err := a.extractor.ExtractText(ctx, text, ctxRefs)
	if err != nil {
		return
	}

	if extraction.SuggestedContextID == nil || *extraction.SuggestedContextID == "" {
		return
	}

	ctxID, err := uuid.Parse(*extraction.SuggestedContextID)
	if err != nil {
		return
	}

	if _, err := a.contextBus.QueryByID(ctx, ctxID); err != nil {
		return
	}

	if extraction.ContextConfidence >= 0.7 {
		note, err := a.noteBus.QueryByID(ctx, entityID)
		if err != nil {
			return
		}
		a.noteBus.Update(ctx, note, notebus.UpdateNote{ContextID: &ctxID}) //nolint:errcheck
	} else {
		busCtxRefs := make([]clarificationbus.ContextRef, len(ctxRefs))
		for i, r := range ctxRefs {
			busCtxRefs[i] = clarificationbus.ContextRef{ID: r.ID, Title: r.Title}
		}
		optJSON, _ := json.Marshal(clarificationbus.ContextAssignmentOptions{
			SuggestedContext:  ctxID.String(),
			Confidence:        extraction.ContextConfidence,
			AvailableContexts: busCtxRefs,
		})
		guess, _ := json.Marshal(map[string]string{"context_id": ctxID.String()})
		guessRaw := json.RawMessage(guess)
		reasoning := fmt.Sprintf("AI matched note to context with %.0f%% confidence", extraction.ContextConfidence*100)

		a.clarificationBus.Create(ctx, clarificationbus.NewClarificationItem{ //nolint:errcheck
			Kind:          clarificationkind.ContextAssignment,
			SubjectType:   entityType,
			SubjectID:     entityID,
			Question:      "Which context does this note belong to?",
			ClaudeGuess:   &guessRaw,
			Reasoning:     &reasoning,
			AnswerOptions: json.RawMessage(optJSON),
		})
	}
}
```

Keep the existing `update`, `delete`, `queryAll`, `queryByID` methods unchanged (do not show them here — only `create` and the new `asyncClassify` method are changed).

- [ ] **Step 2: Update `app/domain/noteapp/route.go` to inject classifier dependencies**

Replace the `Add` method:

```go
func (Routes) Add(a *web.App, cfg mux.Config) {
	noteStore := notedb.NewStore(cfg.Log, cfg.DB)
	noteBus := notebus.NewBusiness(cfg.Log, noteStore)

	ctxStore := contextdb.NewStore(cfg.Log, cfg.DB)
	ctxBus := contextbus.NewBusiness(cfg.Log, ctxStore)

	clStore := clarificationdb.NewStore(cfg.Log, cfg.DB)
	clBus := clarificationbus.NewBusiness(cfg.Log, clStore)

	claudeExt := extractor.NewClaudeCodeExtractor(cfg.ClaudeCLI)
	var ext extractor.Extractor = claudeExt
	if cfg.OllamaEnabled && cfg.OllamaURL != "" {
		ollamaExt := extractor.NewOllamaExtractor(cfg.OllamaURL, cfg.OllamaModel)
		ext = extractor.NewFailoverExtractor(cfg.Log, claudeExt, ollamaExt)
	}

	hdl := &app{
		noteBus:          noteBus,
		contextBus:       ctxBus,
		clarificationBus: clBus,
		extractor:        ext,
	}
	authen := mid.Auth(cfg.APIKey)

	a.Handle(http.MethodGet, "/api/v1/notes", hdl.queryAll, authen)
	a.Handle(http.MethodGet, "/api/v1/notes/{note_id}", hdl.queryByID, authen)
	a.Handle(http.MethodPost, "/api/v1/notes", hdl.create, authen)
	a.Handle(http.MethodPut, "/api/v1/notes/{note_id}", hdl.update, authen)
	a.Handle(http.MethodDelete, "/api/v1/notes/{note_id}", hdl.delete, authen)
}
```

Add imports to `route.go` that were previously not needed:
```go
"github.com/casebrophy/planner/business/domain/clarificationbus"
"github.com/casebrophy/planner/business/domain/clarificationbus/stores/clarificationdb"
"github.com/casebrophy/planner/business/domain/contextbus"
"github.com/casebrophy/planner/business/domain/contextbus/stores/contextdb"
"github.com/casebrophy/planner/business/domain/ingestbus/extractor"
```

- [ ] **Step 3: Repeat the same pattern for `app/domain/eventapp/eventapp.go` and `eventapp/route.go`**

The pattern is identical to noteapp. In `eventapp.go`:
- Add `contextBus`, `clarificationBus`, `extractor` fields to `app` struct
- After `eventBus.Create()`, if `event.ContextID == nil`, fire `go a.asyncClassify(...)` with text `fmt.Sprintf("Event: %s\nDescription: %s", event.Title, event.Description)`
- Add the same `asyncClassify` method (change "note" references to "event", use `eventBus.Update` with `eventbus.UpdateEvent{ContextID: &ctxID}`)

In `eventapp/route.go`:
- Inject `contextbus`, `clarificationbus`, `extractor` the same way as noteapp

- [ ] **Step 4: Build and test**

```bash
go build ./app/domain/noteapp/... ./app/domain/eventapp/...
make test
```

Expected: no errors, all tests pass

- [ ] **Step 5: Commit**

```bash
git add app/domain/noteapp/ app/domain/eventapp/
git commit -m "feat: auto-classify notes and events on create (async background goroutine)"
```

---

## Phase 4: Frontend

### Task 11: TypeScript types for entity links

**Files:**
- Create: `web/src/types/entityLink.ts`
- Modify: `web/src/types/index.ts`

- [ ] **Step 1: Create `web/src/types/entityLink.ts`**

```typescript
export type EntityKind = 'task' | 'note' | 'event'
export type LinkKind = 'manual' | 'ai_suggested'

export interface EntityLink {
  id: string
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
  confidence: number
  kind: LinkKind
  createdAt: string
}

export interface NewEntityLink {
  sourceType: EntityKind
  sourceId: string
  targetType: EntityKind
  targetId: string
}

export interface EntityLinkFilter {
  entityType?: EntityKind
  entityId?: string
}
```

- [ ] **Step 2: Re-export from `web/src/types/index.ts`**

Add to the existing re-exports in `index.ts`:

```typescript
export type { EntityLink, NewEntityLink, EntityLinkFilter, EntityKind, LinkKind } from './entityLink'
```

- [ ] **Step 3: Verify TypeScript compiles**

```bash
make frontend-build
```

Expected: no type errors

- [ ] **Step 4: Commit**

```bash
git add web/src/types/entityLink.ts web/src/types/index.ts
git commit -m "feat: add EntityLink TypeScript types"
```

---

### Task 12: entityLinkService and entityLinkStore

**Files:**
- Create: `web/src/services/entityLinkService.ts`
- Create: `web/src/stores/entityLinkStore.ts`

- [ ] **Step 1: Create `web/src/services/entityLinkService.ts`**

The CRUD factory doesn't model the query-by-entity pattern well (it uses `GET /entity-links?entity_type=X&entity_id=Y`). Write a focused custom service:

```typescript
import { request } from './client'
import type { EntityLink, NewEntityLink } from '@/types'

export const entityLinkService = {
  async listByEntity(entityType: string, entityId: string): Promise<EntityLink[]> {
    const result = await request<{ items: EntityLink[]; total: number }>(
      '/api/v1/entity-links',
      { params: { entity_type: entityType, entity_id: entityId } },
    )
    return result.items
  },

  async create(link: NewEntityLink): Promise<EntityLink> {
    return request<EntityLink>('/api/v1/entity-links', { method: 'POST', body: link })
  },

  async delete(id: string): Promise<void> {
    return request<void>(`/api/v1/entity-links/${id}`, { method: 'DELETE' })
  },
}
```

- [ ] **Step 2: Create `web/src/stores/entityLinkStore.ts`**

```typescript
import { ref } from 'vue'
import { defineStore } from 'pinia'
import { entityLinkService } from '@/services/entityLinkService'
import { useToastStore } from './toastStore'
import type { EntityLink, NewEntityLink } from '@/types'

export const useEntityLinkStore = defineStore('entityLink', () => {
  // Keyed by `${entityType}:${entityId}` → EntityLink[]
  const linksByEntity = ref<Record<string, EntityLink[]>>({})
  const loading = ref(false)

  function cacheKey(entityType: string, entityId: string): string {
    return `${entityType}:${entityId}`
  }

  async function fetchLinks(entityType: string, entityId: string, force = false): Promise<void> {
    const key = cacheKey(entityType, entityId)
    if (!force && linksByEntity.value[key] !== undefined) return

    loading.value = true
    try {
      linksByEntity.value[key] = await entityLinkService.listByEntity(entityType, entityId)
    } catch (e) {
      useToastStore().error('Failed to load related items')
    } finally {
      loading.value = false
    }
  }

  function getLinks(entityType: string, entityId: string): EntityLink[] {
    return linksByEntity.value[cacheKey(entityType, entityId)] ?? []
  }

  async function createLink(link: NewEntityLink): Promise<EntityLink | null> {
    try {
      const created = await entityLinkService.create(link)
      // Invalidate cache for both sides
      delete linksByEntity.value[cacheKey(link.sourceType, link.sourceId)]
      delete linksByEntity.value[cacheKey(link.targetType, link.targetId)]
      return created
    } catch (e) {
      useToastStore().error('Failed to create link')
      return null
    }
  }

  async function deleteLink(link: EntityLink): Promise<void> {
    try {
      await entityLinkService.delete(link.id)
      // Invalidate cache for both sides
      delete linksByEntity.value[cacheKey(link.sourceType, link.sourceId)]
      delete linksByEntity.value[cacheKey(link.targetType, link.targetId)]
    } catch (e) {
      useToastStore().error('Failed to remove link')
    }
  }

  return { linksByEntity, loading, fetchLinks, getLinks, createLink, deleteLink }
})
```

- [ ] **Step 3: Write service test**

Create `web/src/__tests__/services/entityLinkService.test.ts`:

```typescript
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { entityLinkService } from '@/services/entityLinkService'

// Mock the client module
vi.mock('@/services/client', () => ({
  request: vi.fn(),
}))

import { request } from '@/services/client'
const mockRequest = request as ReturnType<typeof vi.fn>

describe('entityLinkService', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('listByEntity passes correct query params', async () => {
    mockRequest.mockResolvedValue({ items: [], total: 0 })
    await entityLinkService.listByEntity('note', 'abc-123')
    expect(mockRequest).toHaveBeenCalledWith('/api/v1/entity-links', {
      params: { entity_type: 'note', entity_id: 'abc-123' },
    })
  })

  it('create posts to correct endpoint', async () => {
    const link = { sourceType: 'note' as const, sourceId: 'a', targetType: 'event' as const, targetId: 'b' }
    mockRequest.mockResolvedValue({ id: 'x', ...link, confidence: 1, kind: 'manual', createdAt: '' })
    await entityLinkService.create(link)
    expect(mockRequest).toHaveBeenCalledWith('/api/v1/entity-links', {
      method: 'POST',
      body: link,
    })
  })

  it('delete calls correct URL', async () => {
    mockRequest.mockResolvedValue(undefined)
    await entityLinkService.delete('link-id-123')
    expect(mockRequest).toHaveBeenCalledWith('/api/v1/entity-links/link-id-123', { method: 'DELETE' })
  })
})
```

- [ ] **Step 4: Run frontend tests**

```bash
make frontend-test
```

Expected: entity link service tests pass

- [ ] **Step 5: Commit**

```bash
git add web/src/services/entityLinkService.ts \
        web/src/stores/entityLinkStore.ts \
        web/src/src/__tests__/services/entityLinkService.test.ts
git commit -m "feat: add entityLinkService and entityLinkStore"
```

---

### Task 13: Related Items panel in NoteDetailView and TaskDetailView

**Files:**
- Modify: `web/src/views/NoteDetailView.vue`
- Modify: `web/src/views/TaskDetailView.vue`

The panel has two sections:
1. **"In same context"** — computed from existing store data (zero backend cost)
2. **"Also related"** — from entityLinkStore

- [ ] **Step 1: Add Related Items panel to `NoteDetailView.vue`**

Find the bottom of the note detail template and add the related items section. First, add the store import to the `<script setup>`:

```typescript
import { useEntityLinkStore } from '@/stores/entityLinkStore'
import { useNoteStore } from '@/stores/noteStore'
import { useTaskStore } from '@/stores/taskStore'

const entityLinkStore = useEntityLinkStore()
const noteStore = useNoteStore()
const taskStore = useTaskStore()

// Load entity links when note is available
watchEffect(async () => {
  if (note.value?.id) {
    await entityLinkStore.fetchLinks('note', note.value.id)
  }
})

// Entities sharing the same context (implicit links via context_id hub)
const inSameContext = computed(() => {
  if (!note.value?.contextId) return []
  const tasks = taskStore.items.filter(t => t.contextId === note.value!.contextId)
  return tasks.map(t => ({ id: t.id, type: 'task' as const, label: t.title }))
})

// Explicit entity links (from entity_links table)
const explicitLinks = computed(() => {
  if (!note.value?.id) return []
  return entityLinkStore.getLinks('note', note.value.id)
})

// New link creation state
const showLinkModal = ref(false)
const linkTargetType = ref<'task' | 'note' | 'event'>('task')
const linkTargetId = ref('')

async function addLink() {
  if (!note.value?.id || !linkTargetId.value) return
  await entityLinkStore.createLink({
    sourceType: 'note',
    sourceId: note.value.id,
    targetType: linkTargetType.value,
    targetId: linkTargetId.value,
  })
  showLinkModal.value = false
  linkTargetId.value = ''
}

async function removeLink(link: import('@/types').EntityLink) {
  await entityLinkStore.deleteLink(link)
}
```

Add to the template (after the note content section):

```html
<!-- Related Items Panel -->
<section class="related-items">
  <h3>Related Items</h3>

  <div v-if="inSameContext.length > 0">
    <h4>In same context</h4>
    <ul>
      <li v-for="item in inSameContext" :key="item.id">
        <RouterLink :to="`/tasks/${item.id}`">{{ item.label }}</RouterLink>
      </li>
    </ul>
  </div>

  <div v-if="explicitLinks.length > 0">
    <h4>Also related</h4>
    <ul>
      <li v-for="link in explicitLinks" :key="link.id">
        <span>{{ link.sourceId === note?.id ? link.targetType : link.sourceType }}:
              {{ link.sourceId === note?.id ? link.targetId : link.sourceId }}</span>
        <button @click="removeLink(link)" aria-label="Remove link">×</button>
      </li>
    </ul>
  </div>

  <button @click="showLinkModal = true">+ Link manually</button>

  <div v-if="showLinkModal" class="link-modal">
    <select v-model="linkTargetType">
      <option value="task">Task</option>
      <option value="note">Note</option>
      <option value="event">Event</option>
    </select>
    <input v-model="linkTargetId" placeholder="Target ID" />
    <button @click="addLink">Link</button>
    <button @click="showLinkModal = false">Cancel</button>
  </div>
</section>
```

> **Note:** The link modal uses a raw ID input for now. In a follow-up, this should be replaced with a search-and-select component. The raw input is acceptable for the initial implementation.

- [ ] **Step 2: Add the same Related Items panel to `TaskDetailView.vue`**

Follow the exact same pattern but:
- Replace `'note'` with `'task'` in `entityLinkStore.fetchLinks` and `createLink`
- Replace `note.value` with `task.value` throughout
- Compute `inSameContext` from `noteStore.items.filter(n => n.contextId === task.value?.contextId)` (notes sharing context, not tasks)

- [ ] **Step 3: Frontend build check**

```bash
make frontend-build
```

Expected: no type errors, no build errors

- [ ] **Step 4: Commit**

```bash
git add web/src/views/NoteDetailView.vue web/src/views/TaskDetailView.vue
git commit -m "feat: add Related Items panel to note and task detail views"
```

---

### Task 14: ClarificationCard entity_link kind handling

**Files:**
- Modify: `web/src/components/clarifications/ClarificationCard.vue`
- Modify: `web/src/types/enums.ts` (or wherever ClarificationKind is defined)

- [ ] **Step 1: Add `EntityLink` to the ClarificationKind enum**

Find the ClarificationKind enum (likely in `web/src/types/enums.ts`):

```bash
grep -rn "ClarificationKind\|entity_link" web/src/types/
```

Add the new kind:

```typescript
export enum ClarificationKind {
  // ... existing values ...
  EntityLink = 'entity_link',
}

export const ClarificationKindLabels: Record<ClarificationKind, string> = {
  // ... existing labels ...
  [ClarificationKind.EntityLink]: 'Link Related Items',
}

export const ClarificationKindColors: Record<ClarificationKind, string> = {
  // ... existing colors ...
  [ClarificationKind.EntityLink]: '#7c3aed',
}
```

- [ ] **Step 2: Add entity_link rendering to `ClarificationCard.vue`**

After the existing `contextAssignmentOptions` computed, add:

```typescript
import type { EntityLinkOptions } from '@/types/generated/clarification-options'

const entityLinkOptions = computed<EntityLinkOptions | null>(() => {
  if (props.item.kind !== ClarificationKind.EntityLink) return null
  return options.value as EntityLinkOptions | null
})
```

Add to the template (after the context assignment section):

```html
<!-- entity_link clarification -->
<div v-if="item.kind === ClarificationKind.EntityLink && entityLinkOptions">
  <p>AI suggests linking:</p>
  <p>
    <strong>{{ entityLinkOptions.source_type }}</strong> →
    <strong>{{ entityLinkOptions.target_type }}</strong>
    ({{ Math.round(entityLinkOptions.confidence * 100) }}% confidence)
  </p>
  <div class="actions">
    <button @click="resolveWithValue({ confirmed: true })">Confirm Link</button>
    <button @click="resolveWithValue({ confirmed: false })">Reject</button>
    <button @click="emit('dismiss')">Dismiss</button>
  </div>
</div>
```

- [ ] **Step 3: Add EntityLinkOptions to the generated types file**

Find `web/src/types/generated/clarification-options.ts` (or equivalent) and add:

```typescript
export interface EntityLinkOptions {
  source_type: string
  source_id: string
  target_type: string
  target_id: string
  confidence: number
}
```

- [ ] **Step 4: Add clarification resolution side-effect for entity_link kind**

In `app/domain/clarificationapp/clarificationapp.go`, add handling for `entity_link` resolution in the resolve handler (alongside the context_assignment handling from Task 9):

```go
if item.Kind == clarificationkind.EntityLink {
    var answer struct {
        Confirmed bool `json:"confirmed"`
    }
    if err := json.Unmarshal([]byte(rc.Answer), &answer); err == nil && answer.Confirmed {
        var opts clarificationbus.EntityLinkOptions
        if err := json.Unmarshal(item.AnswerOptions, &opts); err == nil {
            sourceID, _ := uuid.Parse(opts.SourceID)
            targetID, _ := uuid.Parse(opts.TargetID)
            a.entityLinkBus.Create(ctx, entitylinkbus.NewEntityLink{ //nolint:errcheck
                SourceType: opts.SourceType,
                SourceID:   sourceID,
                TargetType: opts.TargetType,
                TargetID:   targetID,
                Confidence: opts.Confidence,
                Kind:       "ai_suggested",
            })
        }
    }
}
```

> This requires adding `entityLinkBus *entitylinkbus.Business` to `clarificationapp.app` and injecting it in `clarificationapp/route.go`.

- [ ] **Step 5: Frontend test for ClarificationCard entity_link kind**

In `web/src/__tests__/components/ClarificationCard.test.ts`, add a test case:

```typescript
it('renders entity_link kind with confirm/reject buttons', () => {
  const item: ClarificationItem = {
    id: 'test-id',
    kind: 'entity_link',
    status: 'pending',
    subjectType: 'note',
    subjectId: 'note-123',
    question: 'Link these items?',
    answerOptions: {
      source_type: 'note',
      source_id: 'note-123',
      target_type: 'event',
      target_id: 'event-456',
      confidence: 0.65,
    },
    createdAt: new Date().toISOString(),
  }

  const wrapper = mount(ClarificationCard, { props: { item } })

  expect(wrapper.text()).toContain('note')
  expect(wrapper.text()).toContain('event')
  expect(wrapper.text()).toContain('65%')
  expect(wrapper.find('button[data-action="confirm"]').exists()).toBe(true)
})
```

> Adjust the button selector to match your component's actual attribute.

- [ ] **Step 6: Build and test**

```bash
make frontend-build
make frontend-test
```

Expected: no errors, ClarificationCard test passes

- [ ] **Step 7: Commit**

```bash
git add web/src/components/clarifications/ClarificationCard.vue \
        web/src/types/enums.ts \
        web/src/types/generated/clarification-options.ts \
        app/domain/clarificationapp/
git commit -m "feat: handle entity_link clarification kind in ClarificationCard + resolution side-effect"
```

---

## Phase 5: Arch Docs Update

### Task 15: Update architecture docs

**Files:**
- Modify: `.docs/arch/classify-backend.md`
- Create: `.docs/arch/entitylink-backend.md`

- [ ] **Step 1: Run go-arch for classify domain**

```bash
# In Claude Code:
/go-arch classify
```

- [ ] **Step 2: Run go-arch for entitylink domain**

```bash
/go-arch entitylink
```

- [ ] **Step 3: Commit**

```bash
git add .docs/arch/
git commit -m "docs: update classify-backend and add entitylink-backend arch maps"
```

---

## Final: Session Close

- [ ] Run all tests: `make test && make frontend-test`
- [ ] Run lints: `make lint && make frontend-lint`
- [ ] Pull, push beads, push git:
  ```bash
  git pull --rebase
  bd dolt push
  git push
  ```
- [ ] Verify: `git status` shows "up to date with origin"
