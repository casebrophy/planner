# Frontend Testing Infrastructure with Service & Store Factories

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract service and store factory patterns to eliminate CRUD duplication, then build a testing infrastructure where the factory is tested once and domain-specific extensions are tested per domain. Composable tests serve as the integration layer, mocking only services while running real stores.

**Architecture:** Two factory functions — `createCRUDService<T, TNew, TUpdate, TFilter>()` for the HTTP layer and `createCRUDStore()` for the Pinia state layer. Each domain composes a factory instance with domain-specific extensions. Tests follow the same split: factory behavior tested once thoroughly, domain tests only cover custom logic. Vitest config formalized with happy-dom environment.

**Tech Stack:** Vitest 2.1.0, @vue/test-utils 2.4.6, happy-dom 15.11.0, Pinia 2.3.0

---

## File Structure

```
web/
  vitest.config.ts                          — NEW: vitest configuration (happy-dom, path aliases)
  src/
    services/
      createCRUDService.ts                  — NEW: service factory function
      taskService.ts                        — MODIFY: use factory + domain filter mapping
      contextService.ts                     — MODIFY: use factory + domain filter mapping + extra methods
      tagService.ts                         — MODIFY: use factory (full CRUD exposed) + association methods
    stores/
      createCRUDStore.ts                    — NEW: store factory function
      taskStore.ts                          — MODIFY: use factory + domain-specific getters
      contextStore.ts                       — MODIFY: use factory + domain-specific getters + events
      tagStore.ts                           — MODIFY: use factory (partial) + association methods
    __tests__/
      services/
        createCRUDService.test.ts           — NEW: factory tests (list, getById, create, update, delete, error mapping, filter mapping)
        taskService.test.ts                 — MODIFY: remove CRUD tests, keep only domain filter params test
        contextService.test.ts              — MODIFY: remove CRUD tests, keep filter params + events tests
        tagService.test.ts                  — MODIFY: remove CRUD tests, keep association method tests
      stores/
        createCRUDStore.test.ts             — NEW: factory tests (fetch, cache, optimistic CRUD, rollback, toast, loading/error)
        taskStore.test.ts                   — NEW: domain-specific getters only (tasksByStatus, overdueCount, hasActiveFilter)
        contextStore.test.ts                — NEW: domain-specific getters + events methods
        tagStore.test.ts                    — NEW: association methods only
      composables/
        useTaskBoard.test.ts                — NEW: integration test (mock service, real store)
        useTaskDetail.test.ts               — NEW: integration test (mock service, real stores)
      helpers/
        mockFetch.ts                        — NEW: shared mock fetch helper
        mockService.ts                      — NEW: factory for mock CRUDService instances
        testFactories.ts                    — NEW: factory functions for test data (makeTask, makeContext, makeTag)
```

## Type Interfaces (for reference throughout plan)

The service factory type signature:

```ts
interface CRUDServiceConfig<TFilter> {
  basePath: string
  mapFilter?: (filter: TFilter) => Record<string, string | number | undefined>
}

interface CRUDService<T, TNew, TUpdate, TFilter> {
  list(params?: ListParams & { filter?: TFilter }): Promise<QueryResult<T>>
  getById(id: string): Promise<T>
  create(item: TNew): Promise<T>
  update(id: string, item: TUpdate): Promise<T>
  delete(id: string): Promise<void>
}
```

The store factory type signature:

```ts
interface CRUDStoreConfig<T, TNew, TUpdate, TFilter> {
  name: string
  service: CRUDService<T, TNew, TUpdate, TFilter>
  defaultOrderBy?: string
  defaultRowsPerPage?: number
}
```

---

### Task 1: Vitest Configuration

**Files:**
- Create: `web/vitest.config.ts`

- [ ] **Step 1: Create vitest config**

```ts
// web/vitest.config.ts
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  test: {
    globals: true,
    environment: 'happy-dom',
    include: ['src/__tests__/**/*.test.ts'],
  },
})
```

- [ ] **Step 2: Verify existing tests still pass**

Run: `cd web && npx vitest run`
Expected: 3 existing service tests pass (taskService, contextService, tagService)

- [ ] **Step 3: Commit**

```bash
git add web/vitest.config.ts
git commit -m "feat(web): add vitest configuration with happy-dom environment"
```

---

### Task 2: Test Helpers (mockFetch + testFactories)

**Files:**
- Create: `web/src/__tests__/helpers/mockFetch.ts`
- Create: `web/src/__tests__/helpers/testFactories.ts`

Note: `mockService.ts` is created in Task 3 alongside `createCRUDService.ts` since it imports the `CRUDService` type.

- [ ] **Step 1: Create mockFetch helper**

Extract the repeated fetch mock pattern from existing tests:

```ts
// web/src/__tests__/helpers/mockFetch.ts
import { vi } from 'vitest'

export function setupMockFetch() {
  const mockFetch = vi.fn()
  vi.stubGlobal('fetch', mockFetch)

  function jsonResponse(data: unknown, status = 200) {
    return Promise.resolve({
      ok: status >= 200 && status < 300,
      status,
      statusText: status === 404 ? 'Not Found' : status === 400 ? 'Bad Request' : 'OK',
      json: () => Promise.resolve(data),
    })
  }

  function noContentResponse() {
    return Promise.resolve({
      ok: true,
      status: 204,
      statusText: 'No Content',
      json: () => Promise.resolve(undefined),
    })
  }

  function networkError() {
    return Promise.reject(new TypeError('Failed to fetch'))
  }

  return { mockFetch, jsonResponse, noContentResponse, networkError }
}
```

- [ ] **Step 2: Create test data factories**

Uses proper enum values for type safety (not string casts):

```ts
// web/src/__tests__/helpers/testFactories.ts
import type { Task, Context, Tag } from '@/types'
import { TaskStatus, TaskPriority, TaskEnergy, ContextStatus } from '@/types'

let counter = 0
function uid(): string {
  return `test-${++counter}-${Date.now()}`
}

export function resetFactoryCounter() {
  counter = 0
}

export function makeTask(overrides: Partial<Task> = {}): Task {
  const id = uid()
  return {
    id,
    title: `Task ${id}`,
    description: '',
    status: TaskStatus.Todo,
    priority: TaskPriority.Medium,
    energy: TaskEnergy.Medium,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeContext(overrides: Partial<Context> = {}): Context {
  const id = uid()
  return {
    id,
    title: `Context ${id}`,
    description: '',
    status: ContextStatus.Active,
    summary: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeTag(overrides: Partial<Tag> = {}): Tag {
  const id = uid()
  return {
    id,
    name: `tag-${id}`,
    ...overrides,
  }
}

export function makeQueryResult<T>(items: T[], total?: number) {
  return {
    items,
    total: total ?? items.length,
    page: 1,
    rowsPerPage: 20,
  }
}
```

- [ ] **Step 3: Verify existing tests still pass (nothing broken by adding helpers)**

Run: `cd web && npx vitest run`
Expected: 3 existing tests pass

- [ ] **Step 4: Commit**

```bash
git add web/src/__tests__/helpers/mockFetch.ts web/src/__tests__/helpers/testFactories.ts
git commit -m "feat(web): add test helpers — mockFetch and testFactories"
```

---

### Task 3: Service Factory — createCRUDService

**Files:**
- Create: `web/src/services/createCRUDService.ts`
- Create: `web/src/__tests__/services/createCRUDService.test.ts`
- Create: `web/src/__tests__/helpers/mockService.ts` (depends on `CRUDService` type from this task)

- [ ] **Step 1: Write the service factory tests**

```ts
// web/src/__tests__/services/createCRUDService.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { createCRUDService } from '@/services/createCRUDService'
import { setupMockFetch } from '../helpers/mockFetch'
import { ApiNotFoundError, ApiValidationError, ApiNetworkError } from '@/types'

interface TestItem {
  id: string
  name: string
}
interface NewTestItem {
  name: string
}
interface UpdateTestItem {
  name?: string
}
interface TestFilter {
  status?: string
  category?: string
}

const { mockFetch, jsonResponse, noContentResponse, networkError } = setupMockFetch()

function createTestService(mapFilter?: (f: TestFilter) => Record<string, string | number | undefined>) {
  return createCRUDService<TestItem, NewTestItem, UpdateTestItem, TestFilter>({
    basePath: '/api/v1/tests',
    mapFilter,
  })
}

beforeEach(() => {
  mockFetch.mockReset()
})

describe('createCRUDService', () => {
  describe('list', () => {
    it('fetches with default params', async () => {
      const data = { items: [], total: 0, page: 1, rowsPerPage: 20 }
      mockFetch.mockReturnValue(jsonResponse(data))

      const service = createTestService()
      const result = await service.list()

      expect(result).toEqual(data)
      expect(mockFetch).toHaveBeenCalledOnce()
      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tests')
    })

    it('passes pagination and ordering params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 2, rowsPerPage: 10 }))

      const service = createTestService()
      await service.list({ page: 2, rows: 10, orderBy: 'name' })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('page=2')
      expect(url).toContain('rows=10')
      expect(url).toContain('orderBy=name')
    })

    it('applies mapFilter to convert domain filter to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      const service = createTestService((f) => ({
        status: f.status,
        cat: f.category,
      }))
      await service.list({ filter: { status: 'active', category: 'work' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).toContain('cat=work')
    })

    it('omits undefined filter values from query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      const service = createTestService((f) => ({
        status: f.status,
        cat: f.category,
      }))
      await service.list({ filter: { status: 'active' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).not.toContain('cat=')
    })
  })

  describe('getById', () => {
    it('fetches a single item by ID', async () => {
      const item = { id: 'abc', name: 'Test' }
      mockFetch.mockReturnValue(jsonResponse(item))

      const service = createTestService()
      const result = await service.getById('abc')

      expect(result).toEqual(item)
      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tests/abc')
    })
  })

  describe('create', () => {
    it('posts a new item', async () => {
      const newItem = { name: 'New' }
      const created = { id: 'new-1', name: 'New' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const service = createTestService()
      const result = await service.create(newItem)

      expect(result).toEqual(created)
      const [, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(options.method).toBe('POST')
      expect(JSON.parse(options.body as string)).toEqual(newItem)
    })
  })

  describe('update', () => {
    it('puts updates to an item', async () => {
      const update = { name: 'Updated' }
      const updated = { id: 'abc', name: 'Updated' }
      mockFetch.mockReturnValue(jsonResponse(updated))

      const service = createTestService()
      const result = await service.update('abc', update)

      expect(result).toEqual(updated)
      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tests/abc')
      expect(options.method).toBe('PUT')
    })
  })

  describe('delete', () => {
    it('deletes an item', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      const service = createTestService()
      await service.delete('abc')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tests/abc')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('error handling', () => {
    it('throws ApiNotFoundError on 404', async () => {
      mockFetch.mockReturnValue(jsonResponse({ error: 'not found' }, 404))

      const service = createTestService()
      await expect(service.getById('missing')).rejects.toThrow(ApiNotFoundError)
    })

    it('throws ApiValidationError on 400', async () => {
      mockFetch.mockReturnValue(
        jsonResponse({ error: 'invalid', fields: { name: 'required' } }, 400),
      )

      const service = createTestService()
      await expect(service.create({ name: '' })).rejects.toThrow(ApiValidationError)
    })

    it('throws ApiNetworkError on fetch failure', async () => {
      mockFetch.mockReturnValue(networkError())

      const service = createTestService()
      await expect(service.list()).rejects.toThrow(ApiNetworkError)
    })
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npx vitest run src/__tests__/services/createCRUDService.test.ts`
Expected: FAIL — `createCRUDService` does not exist yet

- [ ] **Step 3: Implement the service factory**

```ts
// web/src/services/createCRUDService.ts
import { request } from './client'
import type { QueryResult, ListParams } from '@/types'

export interface CRUDServiceConfig<TFilter> {
  basePath: string
  mapFilter?: (filter: TFilter) => Record<string, string | number | undefined>
}

export interface CRUDService<T, TNew, TUpdate, TFilter> {
  list(params?: ListParams & { filter?: TFilter }): Promise<QueryResult<T>>
  getById(id: string): Promise<T>
  create(item: TNew): Promise<T>
  update(id: string, item: TUpdate): Promise<T>
  delete(id: string): Promise<void>
}

export function createCRUDService<T, TNew, TUpdate, TFilter = Record<string, never>>(
  config: CRUDServiceConfig<TFilter>,
): CRUDService<T, TNew, TUpdate, TFilter> {
  const { basePath, mapFilter } = config

  return {
    async list(params: ListParams & { filter?: TFilter } = {}): Promise<QueryResult<T>> {
      const queryParams: Record<string, string | number | undefined> = {
        page: params.page,
        rows: params.rows,
        orderBy: params.orderBy,
      }

      if (params.filter && mapFilter) {
        Object.assign(queryParams, mapFilter(params.filter))
      }

      return request<QueryResult<T>>(basePath, { params: queryParams })
    },

    async getById(id: string): Promise<T> {
      return request<T>(`${basePath}/${id}`)
    },

    async create(item: TNew): Promise<T> {
      return request<T>(basePath, { method: 'POST', body: item })
    },

    async update(id: string, item: TUpdate): Promise<T> {
      return request<T>(`${basePath}/${id}`, { method: 'PUT', body: item })
    },

    async delete(id: string): Promise<void> {
      return request<void>(`${basePath}/${id}`, { method: 'DELETE' })
    },
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npx vitest run src/__tests__/services/createCRUDService.test.ts`
Expected: All 9 tests PASS

- [ ] **Step 5: Create mockService helper** (depends on CRUDService type just created)

```ts
// web/src/__tests__/helpers/mockService.ts
import { vi } from 'vitest'
import type { CRUDService } from '@/services/createCRUDService'

export function createMockService<T, TNew, TUpdate, TFilter>(): CRUDService<T, TNew, TUpdate, TFilter> & {
  list: ReturnType<typeof vi.fn>
  getById: ReturnType<typeof vi.fn>
  create: ReturnType<typeof vi.fn>
  update: ReturnType<typeof vi.fn>
  delete: ReturnType<typeof vi.fn>
} {
  return {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  }
}
```

- [ ] **Step 6: Commit**

```bash
git add web/src/services/createCRUDService.ts web/src/__tests__/services/createCRUDService.test.ts web/src/__tests__/helpers/mockService.ts
git commit -m "feat(web): add createCRUDService factory with full test coverage"
```

---

### Task 4: Migrate taskService to Factory

**Files:**
- Modify: `web/src/services/taskService.ts`
- Modify: `web/src/__tests__/services/taskService.test.ts`

- [ ] **Step 1: Rewrite taskService to use factory**

```ts
// web/src/services/taskService.ts
import { createCRUDService } from './createCRUDService'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'

export const taskService = createCRUDService<Task, NewTask, UpdateTask, TaskFilter>({
  basePath: '/api/v1/tasks',
  mapFilter: (f) => ({
    status: f.status,
    priority: f.priority,
    context_id: f.contextId,
    start_due_date: f.startDueDate,
    end_due_date: f.endDueDate,
  }),
})
```

- [ ] **Step 2: Rewrite taskService test — only domain-specific filter mapping**

```ts
// web/src/__tests__/services/taskService.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { taskService } from '@/services/taskService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('taskService', () => {
  describe('list filter mapping', () => {
    it('maps TaskFilter fields to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      await taskService.list({
        page: 1,
        rows: 20,
        filter: {
          status: 'todo',
          priority: 'high',
          contextId: 'ctx-1',
          startDueDate: '2026-01-01',
          endDueDate: '2026-12-31',
        },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=todo')
      expect(url).toContain('priority=high')
      expect(url).toContain('context_id=ctx-1')
      expect(url).toContain('start_due_date=2026-01-01')
      expect(url).toContain('end_due_date=2026-12-31')
    })

    it('omits undefined filter fields', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 20 }))

      await taskService.list({ filter: { status: 'todo' } })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=todo')
      expect(url).not.toContain('priority')
      expect(url).not.toContain('context_id')
    })
  })
})
```

- [ ] **Step 3: Run tests**

Run: `cd web && npx vitest run src/__tests__/services/taskService.test.ts`
Expected: PASS

- [ ] **Step 4: Run full test suite + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: All pass — no regressions

- [ ] **Step 5: Commit**

```bash
git add web/src/services/taskService.ts web/src/__tests__/services/taskService.test.ts
git commit -m "refactor(web): migrate taskService to createCRUDService factory"
```

---

### Task 5: Migrate contextService to Factory

**Files:**
- Modify: `web/src/services/contextService.ts`
- Modify: `web/src/__tests__/services/contextService.test.ts`

- [ ] **Step 1: Rewrite contextService to use factory + keep extra methods**

```ts
// web/src/services/contextService.ts
import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type {
  Context,
  NewContext,
  UpdateContext,
  ContextFilter,
  ContextEvent,
  NewEvent,
  QueryResult,
  ListParams,
} from '@/types'

const crud = createCRUDService<Context, NewContext, UpdateContext, ContextFilter>({
  basePath: '/api/v1/contexts',
  mapFilter: (f) => ({
    status: f.status,
    title: f.title,
  }),
})

export const contextService = {
  ...crud,

  async listEvents(
    contextId: string,
    params: ListParams = {},
  ): Promise<QueryResult<ContextEvent>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
    }
    return request<QueryResult<ContextEvent>>(`/api/v1/contexts/${contextId}/events`, {
      params: queryParams,
    })
  },

  async addEvent(contextId: string, event: NewEvent): Promise<ContextEvent> {
    return request<ContextEvent>(`/api/v1/contexts/${contextId}/events`, {
      method: 'POST',
      body: event,
    })
  },
}
```

- [ ] **Step 2: Rewrite contextService test — filter mapping + extra methods only**

```ts
// web/src/__tests__/services/contextService.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { contextService } from '@/services/contextService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('contextService', () => {
  describe('list filter mapping', () => {
    it('maps ContextFilter fields to query params', async () => {
      mockFetch.mockReturnValue(jsonResponse({ items: [], total: 0, page: 1, rowsPerPage: 50 }))

      await contextService.list({
        filter: { status: 'active', title: 'Project' },
      })

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('status=active')
      expect(url).toContain('title=Project')
    })
  })

  describe('listEvents', () => {
    it('fetches events for a context', async () => {
      const data = { items: [], total: 0, page: 1, rowsPerPage: 50 }
      mockFetch.mockReturnValue(jsonResponse(data))

      const result = await contextService.listEvents('ctx-1', { page: 1, rows: 50 })
      expect(result).toEqual(data)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts/ctx-1/events')
    })
  })

  describe('addEvent', () => {
    it('posts a new event', async () => {
      const event = { kind: 'note', body: 'test' }
      const created = { id: 'evt-1', ...event, createdAt: '2026-01-01' }
      mockFetch.mockReturnValue(jsonResponse(created))

      const result = await contextService.addEvent('ctx-1', event as any)
      expect(result).toEqual(created)

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/events')
      expect(options.method).toBe('POST')
    })
  })
})
```

- [ ] **Step 3: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add web/src/services/contextService.ts web/src/__tests__/services/contextService.test.ts
git commit -m "refactor(web): migrate contextService to createCRUDService factory"
```

---

### Task 6: Migrate tagService to Factory

**Files:**
- Modify: `web/src/services/tagService.ts`
- Modify: `web/src/__tests__/services/tagService.test.ts`

- [ ] **Step 1: Rewrite tagService — full factory CRUD + association methods**

tagService uses the factory for all five CRUD methods (even though `getById` and `update` aren't used yet — this satisfies the `CRUDService` interface so the store factory can accept it). Association methods are domain-specific additions.

```ts
// web/src/services/tagService.ts
import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { Tag, NewTag } from '@/types'

const crud = createCRUDService<Tag, NewTag, Partial<Tag>, Record<string, never>>({
  basePath: '/api/v1/tags',
})

export const tagService = {
  ...crud,

  async getByTask(taskId: string): Promise<Tag[]> {
    return request<Tag[]>(`/api/v1/tasks/${taskId}/tags`)
  },

  async addToTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromTask(taskId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${taskId}/tags/${tagId}`, { method: 'DELETE' })
  },

  async getByContext(contextId: string): Promise<Tag[]> {
    return request<Tag[]>(`/api/v1/contexts/${contextId}/tags`)
  },

  async addToContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'POST' })
  },

  async removeFromContext(contextId: string, tagId: string): Promise<void> {
    return request<void>(`/api/v1/contexts/${contextId}/tags/${tagId}`, { method: 'DELETE' })
  },
}
```

- [ ] **Step 2: Rewrite tagService test — association methods only**

```ts
// web/src/__tests__/services/tagService.test.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { tagService } from '@/services/tagService'
import { setupMockFetch } from '../helpers/mockFetch'

const { mockFetch, jsonResponse, noContentResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

describe('tagService', () => {
  describe('getByTask', () => {
    it('fetches tags for a task', async () => {
      const tags = [{ id: 't1', name: 'urgent' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByTask('task-1')
      expect(result).toEqual(tags)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/tasks/task-1/tags')
    })
  })

  describe('addToTask', () => {
    it('posts tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.addToTask('task-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/task-1/tags/tag-1')
      expect(options.method).toBe('POST')
    })
  })

  describe('removeFromTask', () => {
    it('deletes tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.removeFromTask('task-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/tasks/task-1/tags/tag-1')
      expect(options.method).toBe('DELETE')
    })
  })

  describe('getByContext', () => {
    it('fetches tags for a context', async () => {
      const tags = [{ id: 't1', name: 'work' }]
      mockFetch.mockReturnValue(jsonResponse(tags))

      const result = await tagService.getByContext('ctx-1')
      expect(result).toEqual(tags)

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/contexts/ctx-1/tags')
    })
  })

  describe('addToContext', () => {
    it('posts tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.addToContext('ctx-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/tags/tag-1')
      expect(options.method).toBe('POST')
    })
  })

  describe('removeFromContext', () => {
    it('deletes tag association', async () => {
      mockFetch.mockReturnValue(noContentResponse())

      await tagService.removeFromContext('ctx-1', 'tag-1')

      const [url, options] = mockFetch.mock.calls[0]! as [string, RequestInit]
      expect(url).toContain('/api/v1/contexts/ctx-1/tags/tag-1')
      expect(options.method).toBe('DELETE')
    })
  })
})
```

- [ ] **Step 3: Run tests + lint + build**

Run: `cd web && npx vitest run && npm run lint && npm run build`
Expected: All pass

- [ ] **Step 4: Commit**

```bash
git add web/src/services/tagService.ts web/src/__tests__/services/tagService.test.ts
git commit -m "refactor(web): migrate tagService to createCRUDService factory"
```

---

### Task 7: Store Factory — createCRUDStore

**Files:**
- Create: `web/src/stores/createCRUDStore.ts`
- Create: `web/src/__tests__/stores/createCRUDStore.test.ts`

- [ ] **Step 1: Write the store factory tests**

These test every behavior of the generic CRUD store: fetching, caching, optimistic create/update/delete, rollback on failure, loading/error state, toast notifications.

```ts
// web/src/__tests__/stores/createCRUDStore.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { defineStore } from 'pinia'
import { createCRUDStore } from '@/stores/createCRUDStore'
import { createMockService } from '../helpers/mockService'
import { makeQueryResult } from '../helpers/testFactories'

interface TestItem {
  id: string
  name: string
}
interface NewTestItem {
  name: string
}
interface UpdateTestItem {
  name?: string
}
interface TestFilter {
  status?: string
}

function makeItem(id: string, name = `Item ${id}`): TestItem {
  return { id, name }
}

let mockService: ReturnType<typeof createMockService<TestItem, NewTestItem, UpdateTestItem, TestFilter>>

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

function createTestStore() {
  return defineStore('test-crud', () => {
    return createCRUDStore<TestItem, NewTestItem, UpdateTestItem, TestFilter>({
      name: 'test item',
      service: mockService,
    })
  })
}

beforeEach(() => {
  setActivePinia(createPinia())
  mockService = createMockService()
})

describe('createCRUDStore', () => {
  describe('fetchList', () => {
    it('fetches items and updates state', async () => {
      const items = [makeItem('1'), makeItem('2')]
      mockService.list.mockResolvedValue(makeQueryResult(items, 2))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList()

      expect(store.items).toEqual(items)
      expect(store.total).toBe(2)
      expect(store.loading).toBe(false)
      expect(store.error).toBeNull()
    })

    it('sets loading state during fetch', async () => {
      let resolvePromise: (value: any) => void
      mockService.list.mockReturnValue(new Promise((r) => { resolvePromise = r }))

      const useStore = createTestStore()
      const store = useStore()

      const promise = store.fetchList(true)
      expect(store.loading).toBe(true)

      resolvePromise!(makeQueryResult([]))
      await promise
      expect(store.loading).toBe(false)
    })

    it('sets error state on failure', async () => {
      mockService.list.mockRejectedValue(new Error('Network error'))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList()

      expect(store.error).toBe('Network error')
      expect(store.items).toEqual([])
    })

    it('passes filter, orderBy, page, rowsPerPage to service', async () => {
      mockService.list.mockResolvedValue(makeQueryResult([]))

      const useStore = createTestStore()
      const store = useStore()
      store.setFilter({ status: 'active' } as TestFilter)
      store.setOrder('name')
      store.setPage(3)

      await store.fetchList(true)

      expect(mockService.list).toHaveBeenCalledWith({
        page: 3,
        rows: 20,
        orderBy: 'name',
        filter: { status: 'active' },
      })
    })

    it('skips fetch when cache is valid', async () => {
      mockService.list.mockResolvedValue(makeQueryResult([]))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList()
      await store.fetchList()

      expect(mockService.list).toHaveBeenCalledTimes(1)
    })

    it('fetches when forced even with valid cache', async () => {
      mockService.list.mockResolvedValue(makeQueryResult([]))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList()
      await store.fetchList(true)

      expect(mockService.list).toHaveBeenCalledTimes(2)
    })
  })

  describe('fetchById', () => {
    it('fetches a single item and sets currentItem', async () => {
      const item = makeItem('1')
      mockService.getById.mockResolvedValue(item)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchById('1')

      expect(store.currentItem).toEqual(item)
    })

    it('sets error on failure', async () => {
      mockService.getById.mockRejectedValue(new Error('Not found'))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchById('missing')

      expect(store.error).toBe('Not found')
    })
  })

  describe('create', () => {
    it('creates item and prepends to list', async () => {
      const existing = makeItem('1')
      const created = makeItem('2', 'New')
      mockService.list.mockResolvedValue(makeQueryResult([existing], 1))
      mockService.create.mockResolvedValue(created)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList(true)
      const result = await store.create({ name: 'New' })

      expect(result).toEqual(created)
      expect(store.items[0]).toEqual(created)
      expect(store.total).toBe(2)
    })

    it('re-throws on failure so composables can handle it', async () => {
      mockService.create.mockRejectedValue(new Error('Validation failed'))

      const useStore = createTestStore()
      const store = useStore()

      await expect(store.create({ name: '' })).rejects.toThrow('Validation failed')
    })
  })

  describe('update (optimistic)', () => {
    it('optimistically updates item in list', async () => {
      const original = makeItem('1', 'Original')
      const updated = makeItem('1', 'Updated')
      mockService.list.mockResolvedValue(makeQueryResult([original], 1))
      mockService.update.mockResolvedValue(updated)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList(true)
      await store.update('1', { name: 'Updated' })

      expect(store.items[0]!.name).toBe('Updated')
    })

    it('optimistically updates currentItem', async () => {
      const original = makeItem('1', 'Original')
      const updated = makeItem('1', 'Updated')
      mockService.getById.mockResolvedValue(original)
      mockService.update.mockResolvedValue(updated)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchById('1')
      await store.update('1', { name: 'Updated' })

      expect(store.currentItem!.name).toBe('Updated')
    })

    it('rolls back on failure', async () => {
      const original = makeItem('1', 'Original')
      mockService.list.mockResolvedValue(makeQueryResult([original], 1))
      mockService.update.mockRejectedValue(new Error('Server error'))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList(true)
      await expect(store.update('1', { name: 'Bad' })).rejects.toThrow('Server error')

      expect(store.items[0]!.name).toBe('Original')
    })

    it('rolls back currentItem on failure', async () => {
      const original = makeItem('1', 'Original')
      mockService.getById.mockResolvedValue(original)
      mockService.update.mockRejectedValue(new Error('Server error'))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchById('1')
      await expect(store.update('1', { name: 'Bad' })).rejects.toThrow('Server error')

      expect(store.currentItem!.name).toBe('Original')
    })
  })

  describe('remove (optimistic)', () => {
    it('optimistically removes item from list', async () => {
      const item = makeItem('1')
      mockService.list.mockResolvedValue(makeQueryResult([item], 1))
      mockService.delete.mockResolvedValue(undefined)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList(true)
      await store.remove('1')

      expect(store.items).toEqual([])
      expect(store.total).toBe(0)
    })

    it('clears currentItem if deleted item matches', async () => {
      const item = makeItem('1')
      mockService.getById.mockResolvedValue(item)
      mockService.delete.mockResolvedValue(undefined)

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchById('1')
      await store.remove('1')

      expect(store.currentItem).toBeNull()
    })

    it('rolls back on failure', async () => {
      const item = makeItem('1')
      mockService.list.mockResolvedValue(makeQueryResult([item], 1))
      mockService.delete.mockRejectedValue(new Error('Cannot delete'))

      const useStore = createTestStore()
      const store = useStore()

      await store.fetchList(true)
      await expect(store.remove('1')).rejects.toThrow('Cannot delete')

      expect(store.items).toEqual([item])
      expect(store.total).toBe(1)
    })
  })

  describe('setFilter', () => {
    it('resets page to 1 when filter changes', () => {
      const useStore = createTestStore()
      const store = useStore()

      store.setPage(5)
      store.setFilter({ status: 'active' } as TestFilter)

      expect(store.page).toBe(1)
    })
  })

  describe('setOrder', () => {
    it('resets page to 1 when order changes', () => {
      const useStore = createTestStore()
      const store = useStore()

      store.setPage(5)
      store.setOrder('name')

      expect(store.page).toBe(1)
      expect(store.orderBy).toBe('name')
    })
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd web && npx vitest run src/__tests__/stores/createCRUDStore.test.ts`
Expected: FAIL — `createCRUDStore` does not exist yet

- [ ] **Step 3: Implement the store factory**

```ts
// web/src/stores/createCRUDStore.ts
import { ref } from 'vue'
import { useToastStore } from './toastStore'
import type { CRUDService } from '@/services/createCRUDService'

const CACHE_TTL = 5 * 60 * 1000

export interface CRUDStoreConfig<T, TNew, TUpdate, TFilter> {
  name: string
  service: CRUDService<T, TNew, TUpdate, TFilter>
  defaultOrderBy?: string
  defaultRowsPerPage?: number
}

export function createCRUDStore<
  T extends { id: string },
  TNew,
  TUpdate,
  TFilter,
>(config: CRUDStoreConfig<T, TNew, TUpdate, TFilter>) {
  const { name, service, defaultOrderBy = 'created_at', defaultRowsPerPage = 20 } = config

  const items = ref<T[]>([]) as { value: T[] }
  const total = ref(0)
  const page = ref(1)
  const rowsPerPage = ref(defaultRowsPerPage)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastFetchedAt = ref<Record<string, number>>({})
  const filter = ref<TFilter>({} as TFilter)
  const orderBy = ref(defaultOrderBy)
  const currentItem = ref<T | null>(null) as { value: T | null }

  const toast = useToastStore()

  function cacheKey(): string {
    return JSON.stringify({ filter: filter.value, orderBy: orderBy.value, page: page.value })
  }

  function isCacheValid(): boolean {
    const key = cacheKey()
    const ts = lastFetchedAt.value[key]
    return ts !== undefined && Date.now() - ts < CACHE_TTL
  }

  async function fetchList(force = false) {
    if (!force && isCacheValid()) return
    loading.value = true
    error.value = null
    try {
      const result = await service.list({
        page: page.value,
        rows: rowsPerPage.value,
        orderBy: orderBy.value,
        filter: filter.value,
      })
      items.value = result.items
      total.value = result.total
      lastFetchedAt.value[cacheKey()] = Date.now()
    } catch (e) {
      error.value = e instanceof Error ? e.message : `Failed to fetch ${name}s`
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchById(id: string) {
    loading.value = true
    error.value = null
    try {
      currentItem.value = await service.getById(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : `Failed to fetch ${name}`
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function create(item: TNew) {
    try {
      const created = await service.create(item)
      items.value.unshift(created)
      total.value++
      toast.success(`${capitalize(name)} created`)
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : `Failed to create ${name}`
      toast.error(msg)
      throw e
    }
  }

  async function update(id: string, data: TUpdate) {
    const idx = items.value.findIndex((item) => item.id === id)
    const backup = idx !== -1 ? { ...items.value[idx]! } : null
    const currentBackup = currentItem.value?.id === id ? { ...currentItem.value } : null

    // Optimistic update
    if (idx !== -1) {
      items.value[idx] = { ...items.value[idx]!, ...stripUndefined(data) }
    }
    if (currentItem.value?.id === id) {
      currentItem.value = { ...currentItem.value, ...stripUndefined(data) }
    }

    try {
      const updated = await service.update(id, data)
      if (idx !== -1) items.value[idx] = updated
      if (currentItem.value?.id === id) currentItem.value = updated
      toast.success(`${capitalize(name)} updated`)
      return updated
    } catch (e) {
      // Rollback
      if (idx !== -1 && backup) items.value[idx] = backup
      if (currentBackup) currentItem.value = currentBackup
      const msg = e instanceof Error ? e.message : `Failed to update ${name}`
      toast.error(msg)
      throw e
    }
  }

  async function remove(id: string) {
    const idx = items.value.findIndex((item) => item.id === id)
    const backup = idx !== -1 ? items.value[idx]! : null

    // Optimistic remove
    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
    }

    try {
      await service.delete(id)
      if (currentItem.value?.id === id) currentItem.value = null
      toast.success(`${capitalize(name)} deleted`)
    } catch (e) {
      // Rollback
      if (backup && idx !== -1) {
        items.value.splice(idx, 0, backup)
        total.value++
      }
      const msg = e instanceof Error ? e.message : `Failed to delete ${name}`
      toast.error(msg)
      throw e
    }
  }

  function setFilter(f: TFilter) {
    filter.value = f
    page.value = 1
  }

  function setPage(p: number) {
    page.value = p
  }

  function setOrder(o: string) {
    orderBy.value = o
    page.value = 1
  }

  return {
    items,
    total,
    page,
    rowsPerPage,
    loading,
    error,
    filter,
    orderBy,
    currentItem,
    lastFetchedAt,
    fetchList,
    fetchById,
    create,
    update,
    remove,
    setFilter,
    setPage,
    setOrder,
  }
}

function stripUndefined(obj: unknown): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    if (value !== undefined) result[key] = value
  }
  return result
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd web && npx vitest run src/__tests__/stores/createCRUDStore.test.ts`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add web/src/stores/createCRUDStore.ts web/src/__tests__/stores/createCRUDStore.test.ts
git commit -m "feat(web): add createCRUDStore factory with full test coverage"
```

---

### ⚠️ Tasks 8–11 are an atomic group

Tasks 8, 9, and 10 rename store methods (e.g., `fetchTasks` → `fetchList`, `currentTask` → `currentItem`). Task 11 updates all composables and views to use the new names. **Between Tasks 8 and 11, `npm run build` will fail** due to broken references. During this window, only run `npx vitest run` on the specific test file being worked on. Full build verification resumes in Task 11 Step 7.

### Task 8: Migrate taskStore to Factory

**Files:**
- Modify: `web/src/stores/taskStore.ts`
- Create: `web/src/__tests__/stores/taskStore.test.ts`

- [ ] **Step 1: Write taskStore tests — domain-specific getters only**

```ts
// web/src/__tests__/stores/taskStore.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createMockService } from '../helpers/mockService'
import { makeTask, makeQueryResult } from '../helpers/testFactories'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'
import { TaskStatus, TaskPriority } from '@/types'

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

// We need to mock the service at module level so the store picks it up
const mockService = createMockService<Task, NewTask, UpdateTask, TaskFilter>()
vi.mock('@/services/taskService', () => ({
  taskService: mockService,
}))

// Import after mocking
const { useTaskStore } = await import('@/stores/taskStore')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useTaskStore (domain-specific)', () => {
  describe('tasksByStatus', () => {
    it('groups tasks by status', async () => {
      const tasks = [
        makeTask({ status: TaskStatus.Todo }),
        makeTask({ status: TaskStatus.InProgress }),
        makeTask({ status: TaskStatus.Todo }),
        makeTask({ status: TaskStatus.Done }),
      ]
      mockService.list.mockResolvedValue(makeQueryResult(tasks))

      const store = useTaskStore()
      await store.fetchList(true)

      expect(store.tasksByStatus['todo']).toHaveLength(2)
      expect(store.tasksByStatus['in_progress']).toHaveLength(1)
      expect(store.tasksByStatus['done']).toHaveLength(1)
    })
  })

  describe('hasActiveFilter', () => {
    it('returns false when no filters set', () => {
      const store = useTaskStore()
      expect(store.hasActiveFilter).toBe(false)
    })

    it('returns true when status filter is set', () => {
      const store = useTaskStore()
      store.setFilter({ status: TaskStatus.Todo })
      expect(store.hasActiveFilter).toBe(true)
    })

    it('returns true when priority filter is set', () => {
      const store = useTaskStore()
      store.setFilter({ priority: TaskPriority.High })
      expect(store.hasActiveFilter).toBe(true)
    })

    it('returns true when contextId filter is set', () => {
      const store = useTaskStore()
      store.setFilter({ contextId: 'ctx-1' })
      expect(store.hasActiveFilter).toBe(true)
    })
  })

  describe('overdueCount', () => {
    it('counts tasks past due that are not done or cancelled', async () => {
      const yesterday = new Date(Date.now() - 86400000).toISOString()
      const tomorrow = new Date(Date.now() + 86400000).toISOString()
      const tasks = [
        makeTask({ dueDate: yesterday, status: TaskStatus.Todo }),
        makeTask({ dueDate: yesterday, status: TaskStatus.InProgress }),
        makeTask({ dueDate: yesterday, status: TaskStatus.Done }),      // not counted
        makeTask({ dueDate: yesterday, status: TaskStatus.Cancelled }), // not counted
        makeTask({ dueDate: tomorrow, status: TaskStatus.Todo }),        // not counted (future)
      ]
      mockService.list.mockResolvedValue(makeQueryResult(tasks))

      const store = useTaskStore()
      await store.fetchList(true)

      expect(store.overdueCount).toBe(2)
    })

    it('returns 0 when no tasks are overdue', async () => {
      const tomorrow = new Date(Date.now() + 86400000).toISOString()
      mockService.list.mockResolvedValue(
        makeQueryResult([makeTask({ dueDate: tomorrow, status: TaskStatus.Todo })]),
      )

      const store = useTaskStore()
      await store.fetchList(true)

      expect(store.overdueCount).toBe(0)
    })
  })
})
```

- [ ] **Step 2: Rewrite taskStore to use factory**

```ts
// web/src/stores/taskStore.ts
import { defineStore } from 'pinia'
import { computed } from 'vue'
import { taskService } from '@/services/taskService'
import { createCRUDStore } from './createCRUDStore'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'
import { TaskStatus } from '@/types'

export const useTaskStore = defineStore('task', () => {
  const crud = createCRUDStore<Task, NewTask, UpdateTask, TaskFilter>({
    name: 'task',
    service: taskService,
    defaultOrderBy: 'created_at',
    defaultRowsPerPage: 20,
  })

  const tasksByStatus = computed(() => {
    const groups: Record<string, Task[]> = {}
    for (const task of crud.items.value) {
      const s = task.status
      if (!groups[s]) groups[s] = []
      groups[s]!.push(task)
    }
    return groups
  })

  const hasActiveFilter = computed(() => {
    const f = crud.filter.value
    return !!(f.status || f.priority || f.contextId)
  })

  const overdueCount = computed(() => {
    const now = new Date()
    return crud.items.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < now &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Cancelled,
    ).length
  })

  return {
    ...crud,
    tasksByStatus,
    hasActiveFilter,
    overdueCount,
  }
})
```

Note: Consumers that previously accessed `currentTask` now access `currentItem`, and `fetchTasks`/`fetchTask` are now `fetchList`/`fetchById`. The composables and views will need updating (Task 11 handles this).

- [ ] **Step 3: Run store tests**

Run: `cd web && npx vitest run src/__tests__/stores/taskStore.test.ts`
Expected: PASS

- [ ] **Step 4: Commit (tests may have other failures due to renamed methods — that's expected and fixed in Task 11)**

```bash
git add web/src/stores/taskStore.ts web/src/__tests__/stores/taskStore.test.ts
git commit -m "refactor(web): migrate taskStore to createCRUDStore factory"
```

---

### Task 9: Migrate contextStore to Factory

**Files:**
- Modify: `web/src/stores/contextStore.ts`
- Create: `web/src/__tests__/stores/contextStore.test.ts`

- [ ] **Step 1: Write contextStore tests — domain-specific getters + events methods**

```ts
// web/src/__tests__/stores/contextStore.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { createMockService } from '../helpers/mockService'
import { makeContext, makeQueryResult } from '../helpers/testFactories'
import type { Context, NewContext, UpdateContext, ContextFilter } from '@/types'
import { ContextStatus } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

const mockCrud = createMockService<Context, NewContext, UpdateContext, ContextFilter>()

// Mock the full contextService including extra methods
const mockContextService = {
  ...mockCrud,
  listEvents: vi.fn(),
  addEvent: vi.fn(),
}

vi.mock('@/services/contextService', () => ({
  contextService: mockContextService,
}))

const { useContextStore } = await import('@/stores/contextStore')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useContextStore (domain-specific)', () => {
  describe('contextsByStatus', () => {
    it('groups contexts by status', async () => {
      const ctxs = [
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Paused }),
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Closed }),
      ]
      mockCrud.list.mockResolvedValue(makeQueryResult(ctxs))

      const store = useContextStore()
      await store.fetchList(true)

      expect(store.contextsByStatus['active']).toHaveLength(2)
      expect(store.contextsByStatus['paused']).toHaveLength(1)
      expect(store.contextsByStatus['closed']).toHaveLength(1)
    })
  })

  describe('status counts', () => {
    it('computes activeCount, pausedCount, closedCount', async () => {
      const ctxs = [
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Paused }),
      ]
      mockCrud.list.mockResolvedValue(makeQueryResult(ctxs))

      const store = useContextStore()
      await store.fetchList(true)

      expect(store.activeCount).toBe(2)
      expect(store.pausedCount).toBe(1)
      expect(store.closedCount).toBe(0)
    })
  })

  describe('fetchEvents', () => {
    it('fetches events for a context', async () => {
      const events = [{ id: 'e1', kind: 'note', body: 'test' }]
      mockContextService.listEvents.mockResolvedValue({ items: events, total: 1, page: 1, rowsPerPage: 50 })

      const store = useContextStore()
      await store.fetchEvents('ctx-1')

      expect(store.events).toEqual(events)
      expect(store.eventsTotal).toBe(1)
    })

    it('toasts error on failure', async () => {
      mockContextService.listEvents.mockRejectedValue(new Error('Failed'))

      const store = useContextStore()
      await store.fetchEvents('ctx-1')

      expect(store.events).toEqual([])
    })
  })

  describe('addEvent', () => {
    it('adds event and prepends to list', async () => {
      const created = { id: 'e1', kind: 'note', body: 'test', createdAt: '2026-01-01' }
      mockContextService.addEvent.mockResolvedValue(created)

      const store = useContextStore()
      const result = await store.addEvent('ctx-1', { kind: 'note', body: 'test' } as any)

      expect(result).toEqual(created)
      expect(store.events[0]).toEqual(created)
    })
  })
})
```

- [ ] **Step 2: Rewrite contextStore to use factory**

```ts
// web/src/stores/contextStore.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { contextService } from '@/services/contextService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { Context, NewContext, UpdateContext, ContextFilter, ContextEvent, NewEvent } from '@/types'
import { ContextStatus } from '@/types'

export const useContextStore = defineStore('context', () => {
  const crud = createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>({
    name: 'context',
    service: contextService,
    defaultOrderBy: 'last_event',
    defaultRowsPerPage: 50,
  })

  const events = ref<ContextEvent[]>([])
  const eventsTotal = ref(0)

  const toast = useToastStore()

  const contextsByStatus = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextStatus.Active]: [],
      [ContextStatus.Paused]: [],
      [ContextStatus.Closed]: [],
    }
    for (const ctx of crud.items.value) {
      groups[ctx.status]?.push(ctx)
    }
    return groups
  })

  const activeCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Active).length)
  const pausedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Paused).length)
  const closedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Closed).length)

  async function fetchEvents(contextId: string, pg = 1) {
    try {
      const result = await contextService.listEvents(contextId, { page: pg, rows: 50 })
      events.value = result.items
      eventsTotal.value = result.total
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch events'
      toast.error(msg)
    }
  }

  async function addEvent(contextId: string, event: NewEvent) {
    try {
      const created = await contextService.addEvent(contextId, event)
      events.value.unshift(created)
      eventsTotal.value++
      toast.success('Event added')
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to add event'
      toast.error(msg)
      throw e
    }
  }

  return {
    ...crud,
    events,
    eventsTotal,
    contextsByStatus,
    activeCount,
    pausedCount,
    closedCount,
    fetchEvents,
    addEvent,
  }
})
```

- [ ] **Step 3: Run store tests**

Run: `cd web && npx vitest run src/__tests__/stores/contextStore.test.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add web/src/stores/contextStore.ts web/src/__tests__/stores/contextStore.test.ts
git commit -m "refactor(web): migrate contextStore to createCRUDStore factory"
```

---

### Task 10: Migrate tagStore to Factory

**Files:**
- Modify: `web/src/stores/tagStore.ts`
- Create: `web/src/__tests__/stores/tagStore.test.ts`

- [ ] **Step 1: Write tagStore tests — association methods only**

```ts
// web/src/__tests__/stores/tagStore.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { makeTag, makeQueryResult } from '../helpers/testFactories'
import type { Tag, NewTag } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

const mockTagService = {
  list: vi.fn(),
  getById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  getByTask: vi.fn(),
  addToTask: vi.fn(),
  removeFromTask: vi.fn(),
  getByContext: vi.fn(),
  addToContext: vi.fn(),
  removeFromContext: vi.fn(),
}

vi.mock('@/services/tagService', () => ({
  tagService: mockTagService,
}))

const { useTagStore } = await import('@/stores/tagStore')

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('useTagStore (domain-specific)', () => {
  describe('fetchTagsForTask', () => {
    it('fetches and stores tags for a task', async () => {
      const tags = [makeTag({ name: 'urgent' }), makeTag({ name: 'work' })]
      mockTagService.getByTask.mockResolvedValue(tags)

      const store = useTagStore()
      const result = await store.fetchTagsForTask('task-1')

      expect(result).toEqual(tags)
      expect(store.taskTags['task-1']).toEqual(tags)
    })

    it('returns empty array on failure', async () => {
      mockTagService.getByTask.mockRejectedValue(new Error('Failed'))

      const store = useTagStore()
      const result = await store.fetchTagsForTask('task-1')

      expect(result).toEqual([])
    })
  })

  describe('addTagToTask', () => {
    it('associates tag and updates local state', async () => {
      const tag = makeTag({ name: 'urgent' })
      mockTagService.list.mockResolvedValue(makeQueryResult([tag]))
      mockTagService.addToTask.mockResolvedValue(undefined)

      const store = useTagStore()
      await store.fetchList(true)
      await store.addTagToTask('task-1', tag.id)

      expect(store.taskTags['task-1']).toContainEqual(tag)
    })
  })

  describe('removeTagFromTask', () => {
    it('removes tag association from local state', async () => {
      const tag = makeTag({ name: 'urgent' })
      mockTagService.getByTask.mockResolvedValue([tag])
      mockTagService.removeFromTask.mockResolvedValue(undefined)

      const store = useTagStore()
      await store.fetchTagsForTask('task-1')
      await store.removeTagFromTask('task-1', tag.id)

      expect(store.taskTags['task-1']).toEqual([])
    })
  })

  describe('fetchTagsForContext', () => {
    it('fetches and stores tags for a context', async () => {
      const tags = [makeTag({ name: 'project' })]
      mockTagService.getByContext.mockResolvedValue(tags)

      const store = useTagStore()
      const result = await store.fetchTagsForContext('ctx-1')

      expect(result).toEqual(tags)
      expect(store.contextTags['ctx-1']).toEqual(tags)
    })
  })

  describe('addTagToContext', () => {
    it('associates tag and updates local state', async () => {
      const tag = makeTag({ name: 'work' })
      mockTagService.list.mockResolvedValue(makeQueryResult([tag]))
      mockTagService.addToContext.mockResolvedValue(undefined)

      const store = useTagStore()
      await store.fetchList(true)
      await store.addTagToContext('ctx-1', tag.id)

      expect(store.contextTags['ctx-1']).toContainEqual(tag)
    })
  })

  describe('removeTagFromContext', () => {
    it('removes tag association from local state', async () => {
      const tag = makeTag({ name: 'work' })
      mockTagService.getByContext.mockResolvedValue([tag])
      mockTagService.removeFromContext.mockResolvedValue(undefined)

      const store = useTagStore()
      await store.fetchTagsForContext('ctx-1')
      await store.removeTagFromContext('ctx-1', tag.id)

      expect(store.contextTags['ctx-1']).toEqual([])
    })
  })
})
```

- [ ] **Step 2: Rewrite tagStore to use factory**

```ts
// web/src/stores/tagStore.ts
import { defineStore } from 'pinia'
import { ref } from 'vue'
import { tagService } from '@/services/tagService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { Tag, NewTag } from '@/types'

export const useTagStore = defineStore('tag', () => {
  const crud = createCRUDStore<Tag, NewTag, Partial<Tag>, Record<string, never>>({
    name: 'tag',
    service: tagService,
    defaultOrderBy: 'name',
    defaultRowsPerPage: 100,
  })

  const taskTags = ref<Record<string, Tag[]>>({})
  const contextTags = ref<Record<string, Tag[]>>({})

  const toast = useToastStore()

  async function fetchTagsForTask(taskId: string) {
    try {
      const tags = await tagService.getByTask(taskId)
      taskTags.value[taskId] = tags
      return tags
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch task tags'
      toast.error(msg)
      return []
    }
  }

  async function addTagToTask(taskId: string, tagId: string) {
    await tagService.addToTask(taskId, tagId)
    const tag = crud.items.value.find((t) => t.id === tagId)
    if (tag) {
      if (!taskTags.value[taskId]) taskTags.value[taskId] = []
      taskTags.value[taskId]!.push(tag)
    }
  }

  async function removeTagFromTask(taskId: string, tagId: string) {
    await tagService.removeFromTask(taskId, tagId)
    if (taskTags.value[taskId]) {
      taskTags.value[taskId] = taskTags.value[taskId]!.filter((t) => t.id !== tagId)
    }
  }

  async function fetchTagsForContext(contextId: string) {
    try {
      const tags = await tagService.getByContext(contextId)
      contextTags.value[contextId] = tags
      return tags
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch context tags'
      toast.error(msg)
      return []
    }
  }

  async function addTagToContext(contextId: string, tagId: string) {
    await tagService.addToContext(contextId, tagId)
    const tag = crud.items.value.find((t) => t.id === tagId)
    if (tag) {
      if (!contextTags.value[contextId]) contextTags.value[contextId] = []
      contextTags.value[contextId]!.push(tag)
    }
  }

  async function removeTagFromContext(contextId: string, tagId: string) {
    await tagService.removeFromContext(contextId, tagId)
    if (contextTags.value[contextId]) {
      contextTags.value[contextId] = contextTags.value[contextId]!.filter((t) => t.id !== tagId)
    }
  }

  return {
    ...crud,
    taskTags,
    contextTags,
    fetchTagsForTask,
    addTagToTask,
    removeTagFromTask,
    fetchTagsForContext,
    addTagToContext,
    removeTagFromContext,
  }
})
```

- [ ] **Step 3: Run store tests**

Run: `cd web && npx vitest run src/__tests__/stores/tagStore.test.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add web/src/stores/tagStore.ts web/src/__tests__/stores/tagStore.test.ts
git commit -m "refactor(web): migrate tagStore to createCRUDStore factory"
```

---

### Task 11: Update Composables for New Store API

The stores now expose `fetchList`/`fetchById`/`currentItem` instead of domain-specific names like `fetchTasks`/`fetchTask`/`currentTask`. Update all composables and views that reference the old names.

**Files:**
- Modify: `web/src/composables/useTaskBoard.ts`
- Modify: `web/src/composables/useTaskDetail.ts`
- Modify: `web/src/composables/useContextBoard.ts`
- Modify: `web/src/composables/useContextDetail.ts`
- Modify: `web/src/composables/useDashboard.ts`
- Modify: Any view files that directly reference store methods

- [ ] **Step 1: Update useTaskBoard**

Replace all `store.fetchTasks` calls with `store.fetchList`:

```ts
// In useTaskBoard.ts, change:
//   store.fetchTasks(true) → store.fetchList(true)
//   store.fetchTasks()     → store.fetchList()
```

- [ ] **Step 2: Update useTaskDetail**

```ts
// In useTaskDetail.ts, change:
//   storeToRefs: currentTask → currentItem
//   taskStore.fetchTask(taskId) → taskStore.fetchById(taskId)
//   taskStore.updateTask(taskId, data) → taskStore.update(taskId, data)
//   taskStore.deleteTask(taskId) → taskStore.remove(taskId)
//   Return: task: currentTask → task: currentItem (keep the `task` key for view compatibility)
```

Full updated file:

```ts
// web/src/composables/useTaskDetail.ts
import { onMounted, computed } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useTagStore } from '@/stores/tagStore'
import { storeToRefs } from 'pinia'
import type { UpdateTask } from '@/types'

export function useTaskDetail(taskId: string) {
  const taskStore = useTaskStore()
  const tagStore = useTagStore()
  const { currentItem, loading } = storeToRefs(taskStore)

  const tags = computed(() => tagStore.taskTags[taskId] ?? [])

  async function load() {
    await Promise.all([taskStore.fetchById(taskId), tagStore.fetchTagsForTask(taskId)])
  }

  async function update(data: UpdateTask) {
    return taskStore.update(taskId, data)
  }

  async function remove() {
    return taskStore.remove(taskId)
  }

  async function addTag(tagId: string) {
    return tagStore.addTagToTask(taskId, tagId)
  }

  async function removeTag(tagId: string) {
    return tagStore.removeTagFromTask(taskId, tagId)
  }

  onMounted(load)

  return {
    task: currentItem,
    tags,
    loading,
    update,
    remove,
    addTag,
    removeTag,
    reload: load,
  }
}
```

- [ ] **Step 3: Update useContextBoard**

Replace `store.fetchContexts` with `store.fetchList`.

- [ ] **Step 4: Update useContextDetail**

Replace `store.fetchContext` → `store.fetchById`, `store.updateContext` → `store.update`, `store.deleteContext` → `store.remove`, `currentContext` → `currentItem`.

- [ ] **Step 5: Update useDashboard**

Check for any references to domain-specific store method names and update.

- [ ] **Step 6: Update views and components**

Known renames in view/component files (verify with grep, there may be more):

| File | Old | New |
|------|-----|-----|
| `src/views/ContextBoardView.vue` | `contextStore.createContext(data)` | `contextStore.create(data)` |
| `src/views/TaskDetailView.vue` | `tagStore.createTag({ name })` | `tagStore.create({ name })` |
| `src/views/ContextDetailView.vue` | `tagStore.createTag({ name })` | `tagStore.create({ name })` |
| `src/components/tags/TagPicker.vue` | `tagStore.fetchTags()` | `tagStore.fetchList()` |
| `src/components/tasks/TaskForm.vue` | `contextStore.fetchContexts()` | `contextStore.fetchList()` |

Run this to find any remaining hits:

```bash
cd web && grep -rn 'fetchTasks\|fetchTask\b\|currentTask\|fetchContexts\|fetchContext\b\|currentContext\|createTask\|updateTask\|deleteTask\|createContext\|updateContext\|deleteContext\|createTag\|deleteTag\|fetchTags' src/views/ src/components/ --include='*.vue' --include='*.ts' | grep -v '__tests__'
```

Update all hits to use the new factory method names.

- [ ] **Step 7: Run lint + build to verify no broken references**

Run: `cd web && npm run lint && npm run build`
Expected: PASS — all type-checked references resolve

- [ ] **Step 8: Commit**

```bash
git add web/src/composables/ web/src/views/ web/src/components/
git commit -m "refactor(web): update composables and views for factory store API"
```

---

### Task 12: Composable Tests — useTaskBoard

**Files:**
- Create: `web/src/__tests__/composables/useTaskBoard.test.ts`

These are the integration tests: mock service, real store, real composable.

- [ ] **Step 1: Write useTaskBoard tests**

```ts
// web/src/__tests__/composables/useTaskBoard.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { makeTask, makeQueryResult } from '../helpers/testFactories'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'
import { TaskStatus } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

const mockService = {
  list: vi.fn(),
  getById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}

vi.mock('@/services/taskService', () => ({
  taskService: mockService,
}))

// Must import after mocks are set up
const { useTaskBoard } = await import('@/composables/useTaskBoard')

// Helper: mount a component that calls the composable so Vue lifecycle hooks fire
let activeWrapper: ReturnType<typeof mount> | null = null

function mountComposable() {
  let result: ReturnType<typeof useTaskBoard>
  const pinia = createPinia()
  const wrapper = mount(
    defineComponent({
      setup() {
        result = useTaskBoard()
        return {}
      },
      template: '<div />',
    }),
    {
      global: {
        plugins: [pinia],
      },
    },
  )
  activeWrapper = wrapper
  return { wrapper, board: result! }
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  // Clean up usePolling intervals and event listeners
  activeWrapper?.unmount()
  activeWrapper = null
})

describe('useTaskBoard', () => {
  it('fetches tasks on mount', async () => {
    const tasks = [makeTask(), makeTask()]
    mockService.list.mockResolvedValue(makeQueryResult(tasks))

    const { board } = mountComposable()
    await nextTick()
    // Wait for the async fetch
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalled())

    expect(board.tasks.value).toEqual(tasks)
    expect(board.total.value).toBe(2)
  })

  it('applies filter and re-fetches', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const { board } = mountComposable()
    await nextTick()

    mockService.list.mockClear()
    board.setFilter({ status: TaskStatus.Todo })
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalled())

    const callArgs = mockService.list.mock.calls[0]![0]
    expect(callArgs.filter).toEqual({ status: 'todo' })
  })

  it('changes page and re-fetches', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const { board } = mountComposable()
    await nextTick()

    mockService.list.mockClear()
    board.setPage(3)
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalled())

    expect(board.page.value).toBe(3)
  })

  it('changes order and re-fetches', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const { board } = mountComposable()
    await nextTick()

    mockService.list.mockClear()
    board.setOrder('title')
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalled())

    expect(board.orderBy.value).toBe('title')
  })

  it('computes isEmpty correctly', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const { board } = mountComposable()
    await nextTick()
    await vi.waitFor(() => expect(board.loading.value).toBe(false))

    expect(board.isEmpty.value).toBe(true)
  })

  it('propagates service errors to error state', async () => {
    mockService.list.mockRejectedValue(new Error('Server down'))

    const { board } = mountComposable()
    await nextTick()
    await vi.waitFor(() => expect(board.loading.value).toBe(false))

    expect(board.error.value).toBe('Server down')
  })

  it('refresh forces re-fetch even with valid cache', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const { board } = mountComposable()
    await nextTick()
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalledTimes(1))

    board.refresh()
    await vi.waitFor(() => expect(mockService.list).toHaveBeenCalledTimes(2))
  })
})
```

- [ ] **Step 2: Run tests**

Run: `cd web && npx vitest run src/__tests__/composables/useTaskBoard.test.ts`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/__tests__/composables/useTaskBoard.test.ts
git commit -m "test(web): add useTaskBoard composable integration tests"
```

---

### Task 13: Composable Tests — useTaskDetail

**Files:**
- Create: `web/src/__tests__/composables/useTaskDetail.test.ts`

- [ ] **Step 1: Write useTaskDetail tests**

```ts
// web/src/__tests__/composables/useTaskDetail.test.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { makeTask, makeTag } from '../helpers/testFactories'
import type { Task, NewTask, UpdateTask, TaskFilter, Tag, NewTag } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

const mockTaskService = {
  list: vi.fn(),
  getById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
}

const mockTagService = {
  list: vi.fn(),
  getById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  getByTask: vi.fn(),
  addToTask: vi.fn(),
  removeFromTask: vi.fn(),
  getByContext: vi.fn(),
  addToContext: vi.fn(),
  removeFromContext: vi.fn(),
}

vi.mock('@/services/taskService', () => ({ taskService: mockTaskService }))
vi.mock('@/services/tagService', () => ({ tagService: mockTagService }))

const { useTaskDetail } = await import('@/composables/useTaskDetail')

let activeWrapper: ReturnType<typeof mount> | null = null

function mountDetail(taskId: string) {
  let result: ReturnType<typeof useTaskDetail>
  const pinia = createPinia()
  const wrapper = mount(
    defineComponent({
      setup() {
        result = useTaskDetail(taskId)
        return {}
      },
      template: '<div />',
    }),
    {
      global: {
        plugins: [pinia],
      },
    },
  )
  activeWrapper = wrapper
  return { wrapper, detail: result! }
}

beforeEach(() => {
  vi.clearAllMocks()
})

afterEach(() => {
  activeWrapper?.unmount()
  activeWrapper = null
})

describe('useTaskDetail', () => {
  it('loads task and tags on mount', async () => {
    const task = makeTask({ id: 'task-1' })
    const tags = [makeTag({ name: 'urgent' })]
    mockTaskService.getById.mockResolvedValue(task)
    mockTagService.getByTask.mockResolvedValue(tags)

    const { detail } = mountDetail('task-1')
    await nextTick()
    await vi.waitFor(() => expect(mockTaskService.getById).toHaveBeenCalledWith('task-1'))

    expect(detail.task.value).toEqual(task)
    expect(detail.tags.value).toEqual(tags)
  })

  it('update calls store.update and returns result', async () => {
    const task = makeTask({ id: 'task-1', title: 'Original' })
    const updated = { ...task, title: 'Updated' }
    mockTaskService.getById.mockResolvedValue(task)
    mockTagService.getByTask.mockResolvedValue([])
    mockTaskService.update.mockResolvedValue(updated)

    const { detail } = mountDetail('task-1')
    await nextTick()
    await vi.waitFor(() => expect(detail.task.value).not.toBeNull())

    const result = await detail.update({ title: 'Updated' })
    expect(result).toEqual(updated)
  })

  it('remove calls store.remove', async () => {
    const task = makeTask({ id: 'task-1' })
    mockTaskService.getById.mockResolvedValue(task)
    mockTagService.getByTask.mockResolvedValue([])
    mockTaskService.delete.mockResolvedValue(undefined)

    const { detail } = mountDetail('task-1')
    await nextTick()
    await vi.waitFor(() => expect(detail.task.value).not.toBeNull())

    await detail.remove()
    expect(mockTaskService.delete).toHaveBeenCalledWith('task-1')
  })

  it('addTag associates tag', async () => {
    const task = makeTask({ id: 'task-1' })
    mockTaskService.getById.mockResolvedValue(task)
    mockTagService.getByTask.mockResolvedValue([])
    mockTagService.addToTask.mockResolvedValue(undefined)

    const { detail } = mountDetail('task-1')
    await nextTick()

    await detail.addTag('tag-1')
    expect(mockTagService.addToTask).toHaveBeenCalledWith('task-1', 'tag-1')
  })

  it('removeTag disassociates tag', async () => {
    const tag = makeTag({ id: 'tag-1' })
    mockTaskService.getById.mockResolvedValue(makeTask({ id: 'task-1' }))
    mockTagService.getByTask.mockResolvedValue([tag])
    mockTagService.removeFromTask.mockResolvedValue(undefined)

    const { detail } = mountDetail('task-1')
    await nextTick()
    await vi.waitFor(() => expect(detail.tags.value).toHaveLength(1))

    await detail.removeTag('tag-1')
    expect(mockTagService.removeFromTask).toHaveBeenCalledWith('task-1', 'tag-1')
    expect(detail.tags.value).toEqual([])
  })

  it('propagates service error on update failure', async () => {
    const task = makeTask({ id: 'task-1' })
    mockTaskService.getById.mockResolvedValue(task)
    mockTagService.getByTask.mockResolvedValue([])
    mockTaskService.update.mockRejectedValue(new Error('Validation failed'))

    const { detail } = mountDetail('task-1')
    await nextTick()
    await vi.waitFor(() => expect(detail.task.value).not.toBeNull())

    await expect(detail.update({ title: '' })).rejects.toThrow('Validation failed')
    // Verify rollback — task should still have original title
    expect(detail.task.value!.title).toBe(task.title)
  })
})
```

- [ ] **Step 2: Run tests**

Run: `cd web && npx vitest run src/__tests__/composables/useTaskDetail.test.ts`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add web/src/__tests__/composables/useTaskDetail.test.ts
git commit -m "test(web): add useTaskDetail composable integration tests"
```

---

### Task 14: Full Suite Verification

- [ ] **Step 1: Run full test suite**

Run: `cd web && npx vitest run`
Expected: All tests pass

- [ ] **Step 2: Run lint**

Run: `cd web && npm run lint`
Expected: No errors

- [ ] **Step 3: Run build**

Run: `cd web && npm run build`
Expected: Clean build with no type errors

- [ ] **Step 4: Review test count**

Run: `cd web && npx vitest run --reporter=verbose 2>&1 | tail -5`

Expected test files:
- `createCRUDService.test.ts` — 9 tests (factory behavior)
- `taskService.test.ts` — 2 tests (filter mapping)
- `contextService.test.ts` — 3 tests (filter mapping + events)
- `tagService.test.ts` — 6 tests (association methods)
- `createCRUDStore.test.ts` — ~16 tests (factory behavior)
- `taskStore.test.ts` — ~6 tests (domain getters)
- `contextStore.test.ts` — ~5 tests (domain getters + events)
- `tagStore.test.ts` — ~6 tests (association methods)
- `useTaskBoard.test.ts` — 7 tests (composable integration)
- `useTaskDetail.test.ts` — 6 tests (composable integration)

Total: ~66 tests

- [ ] **Step 5: Final commit (if any cleanup was needed)**

```bash
git add -A
git commit -m "chore(web): frontend testing infrastructure complete"
```
