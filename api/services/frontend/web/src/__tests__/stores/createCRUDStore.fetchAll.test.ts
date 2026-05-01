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

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({
    success: vi.fn(),
    error: vi.fn(),
    info: vi.fn(),
    dismiss: vi.fn(),
  }),
}))

function createTestStore() {
  return defineStore('test-crud-fetchall', () => {
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

describe('createCRUDStore.fetchAll', () => {
  it('fetches all pages and accumulates items', async () => {
    const page1 = Array.from({ length: 100 }, (_, i) => makeItem(`1-${i}`))
    const page2 = Array.from({ length: 50 }, (_, i) => makeItem(`2-${i}`))

    mockService.list
      .mockResolvedValueOnce(makeQueryResult(page1, 150))
      .mockResolvedValueOnce(makeQueryResult(page2, 150))

    const useStore = createTestStore()
    const store = useStore()

    await store.fetchAll()

    expect(store.items).toHaveLength(150)
    expect(store.total).toBe(150)
    expect(store.loading).toBe(false)
    expect(mockService.list).toHaveBeenCalledTimes(2)
  })

  it('caches results and skips refetch without force', async () => {
    const items = Array.from({ length: 50 }, (_, i) => makeItem(`${i}`))
    mockService.list.mockResolvedValue(makeQueryResult(items, 50))

    const useStore = createTestStore()
    const store = useStore()

    await store.fetchAll()
    expect(mockService.list).toHaveBeenCalledTimes(1)

    await store.fetchAll()
    expect(mockService.list).toHaveBeenCalledTimes(1)
  })

  it('refetches when force is true', async () => {
    const items = Array.from({ length: 50 }, (_, i) => makeItem(`${i}`))
    mockService.list.mockResolvedValue(makeQueryResult(items, 50))

    const useStore = createTestStore()
    const store = useStore()

    await store.fetchAll()
    expect(mockService.list).toHaveBeenCalledTimes(1)

    await store.fetchAll(undefined, true)
    expect(mockService.list).toHaveBeenCalledTimes(2)
  })

  it('merges filter override with store filter', async () => {
    mockService.list.mockResolvedValue(makeQueryResult([]))

    const useStore = createTestStore()
    const store = useStore()
    store.filter = { status: 'open' } as TestFilter

    await store.fetchAll({ status: 'closed' })

    expect(mockService.list).toHaveBeenCalledWith(
      expect.objectContaining({
        filter: { status: 'closed' },
      }),
    )
  })

  it('sets loading state during fetch', async () => {
    let resolvePromise: (value: ReturnType<typeof makeQueryResult>) => void
    mockService.list.mockReturnValue(new Promise((r) => { resolvePromise = r }))

    const useStore = createTestStore()
    const store = useStore()

    const promise = store.fetchAll()
    expect(store.loading).toBe(true)

    resolvePromise!(makeQueryResult([]))
    await promise
    expect(store.loading).toBe(false)
  })

  it('sets error state on failure', async () => {
    mockService.list.mockRejectedValue(new Error('Network error'))

    const useStore = createTestStore()
    const store = useStore()

    await store.fetchAll()

    expect(store.error).toBe('Network error')
  })
})
