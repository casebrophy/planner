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
      let resolvePromise: (value: ReturnType<typeof makeQueryResult>) => void
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
