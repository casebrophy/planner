import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTimeBlockStore } from '@/stores/timeBlockStore'
import { makeTimeBlock, makeQueryResult } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/timeBlockService', () => ({
  timeBlockService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

import { timeBlockService } from '@/services/timeBlockService'

describe('timeBlockStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('CRUD operations', () => {
    it('fetchList calls service.list with correct params', async () => {
      vi.mocked(timeBlockService.list).mockResolvedValue(
        makeQueryResult([], 0)
      )

      const store = useTimeBlockStore()
      await store.fetchList(true)

      expect(timeBlockService.list).toHaveBeenCalledWith({
        page: 1,
        rows: 50,
        orderBy: 'starts_at',
        filter: {},
      })
    })

    it('fetchList populates items and total', async () => {
      const blocks = [makeTimeBlock(), makeTimeBlock()]
      vi.mocked(timeBlockService.list).mockResolvedValue(
        makeQueryResult(blocks, 2)
      )

      const store = useTimeBlockStore()
      await store.fetchList(true)

      expect(store.items).toEqual(blocks)
      expect(store.total).toBe(2)
    })

    it('fetchById fetches a single time block', async () => {
      const block = makeTimeBlock()
      vi.mocked(timeBlockService.getById).mockResolvedValue(block)

      const store = useTimeBlockStore()
      await store.fetchById(block.id)

      expect(timeBlockService.getById).toHaveBeenCalledWith(block.id)
      expect(store.currentItem).toEqual(block)
    })

    it('create adds item to store and returns created item', async () => {
      const newBlock = makeTimeBlock()
      vi.mocked(timeBlockService.create).mockResolvedValue(newBlock)

      const store = useTimeBlockStore()
      const result = await store.create({
        taskId: newBlock.taskId,
        startsAt: newBlock.startsAt,
        endsAt: newBlock.endsAt,
      })

      expect(result).toEqual(newBlock)
      expect(store.items).toContainEqual(newBlock)
      expect(store.items[0]).toEqual(newBlock) // Added at beginning
      expect(store.total).toBe(1)
    })

    it('update modifies item in store optimistically', async () => {
      const original = makeTimeBlock()
      const updated = { ...original, confirmed: true }
      vi.mocked(timeBlockService.update).mockResolvedValue(updated)

      const store = useTimeBlockStore()
      store.items = [original]

      await store.update(original.id, { confirmed: true })

      expect(timeBlockService.update).toHaveBeenCalledWith(original.id, { confirmed: true })
      expect(store.items[0]).toEqual(updated)
    })

    it('remove deletes item from store and calls service', async () => {
      const block = makeTimeBlock()
      vi.mocked(timeBlockService.delete).mockResolvedValue(undefined)

      const store = useTimeBlockStore()
      store.items = [block]
      store.total = 1

      await store.remove(block.id)

      expect(timeBlockService.delete).toHaveBeenCalledWith(block.id)
      expect(store.items).not.toContainEqual(block)
      expect(store.total).toBe(0)
    })
  })

  describe('defaults and configuration', () => {
    it('defaults to starts_at ordering', () => {
      const store = useTimeBlockStore()
      expect(store.orderBy).toBe('starts_at')
    })

    it('defaults to 50 rows per page', () => {
      const store = useTimeBlockStore()
      expect(store.rowsPerPage).toBe(50)
    })

    it('initializes with empty items array', () => {
      const store = useTimeBlockStore()
      expect(store.items).toEqual([])
    })

    it('initializes with zero total', () => {
      const store = useTimeBlockStore()
      expect(store.total).toBe(0)
    })
  })

  describe('filter and pagination', () => {
    it('setFilter updates filter and resets page to 1', async () => {
      vi.mocked(timeBlockService.list).mockResolvedValue(makeQueryResult([], 0))

      const store = useTimeBlockStore()
      store.page = 3

      store.setFilter({ taskId: 'task-1' })

      expect(store.filter).toEqual({ taskId: 'task-1' })
      expect(store.page).toBe(1)
    })

    it('setPage updates current page', () => {
      const store = useTimeBlockStore()
      store.setPage(2)

      expect(store.page).toBe(2)
    })

    it('setOrder updates orderBy and resets page to 1', () => {
      const store = useTimeBlockStore()
      store.page = 3

      store.setOrder('ends_at')

      expect(store.orderBy).toBe('ends_at')
      expect(store.page).toBe(1)
    })
  })

  describe('error handling', () => {
    it('sets error and shows toast on fetchList failure', async () => {
      const error = new Error('Network error')
      vi.mocked(timeBlockService.list).mockRejectedValue(error)

      const store = useTimeBlockStore()
      await store.fetchList(true)

      expect(store.error).toBe('Network error')
      expect(store.loading).toBe(false)
    })

    it('rollback on update failure', async () => {
      const original = makeTimeBlock()
      vi.mocked(timeBlockService.update).mockRejectedValue(new Error('Update failed'))

      const store = useTimeBlockStore()
      store.items = [original]

      await expect(store.update(original.id, { confirmed: true })).rejects.toThrow()

      expect(store.items[0]).toEqual(original) // Restored
    })

    it('rollback on remove failure', async () => {
      const block = makeTimeBlock()
      vi.mocked(timeBlockService.delete).mockRejectedValue(new Error('Delete failed'))

      const store = useTimeBlockStore()
      store.items = [block]
      store.total = 1

      await expect(store.remove(block.id)).rejects.toThrow()

      expect(store.items).toContainEqual(block) // Restored
      expect(store.total).toBe(1)
    })
  })
})
