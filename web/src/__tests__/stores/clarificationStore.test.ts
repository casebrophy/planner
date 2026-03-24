import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useClarificationStore } from '@/stores/clarificationStore'
import { makeClarificationItem } from '../helpers/testFactories'
import { clarificationService } from '@/services/clarificationService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/clarificationService', () => ({
  clarificationService: {
    queryQueue: vi.fn(),
    countPending: vi.fn(),
    resolve: vi.fn(),
    snooze: vi.fn(),
    dismiss: vi.fn(),
  },
}))

describe('clarificationStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchQueue', () => {
    it('populates items and total', async () => {
      const items = [makeClarificationItem(), makeClarificationItem()]
      vi.mocked(clarificationService.queryQueue).mockResolvedValue({
        items,
        total: 2,
        page: 1,
        rowsPerPage: 50,
      })

      const store = useClarificationStore()
      await store.fetchQueue()

      expect(store.items).toHaveLength(2)
      expect(store.total).toBe(2)
      expect(store.currentIndex).toBe(0)
    })
  })

  describe('currentItem', () => {
    it('returns item at currentIndex', () => {
      const store = useClarificationStore()
      const item1 = makeClarificationItem()
      const item2 = makeClarificationItem()
      store.items = [item1, item2]
      store.currentIndex = 1
      expect(store.currentItem).toEqual(item2)
    })

    it('returns null when items empty', () => {
      const store = useClarificationStore()
      expect(store.currentItem).toBeNull()
    })
  })

  describe('resolve', () => {
    it('removes item and decrements pendingCount', async () => {
      const item = makeClarificationItem()
      vi.mocked(clarificationService.resolve).mockResolvedValue(item)

      const store = useClarificationStore()
      store.items = [item]
      store.total = 1
      store.pendingCount = 1

      await store.resolve(item.id, { action: 'confirm' })

      expect(store.items).toHaveLength(0)
      expect(store.pendingCount).toBe(0)
    })
  })

  describe('snooze', () => {
    it('removes item and decrements pendingCount', async () => {
      const item = makeClarificationItem()
      vi.mocked(clarificationService.snooze).mockResolvedValue(item)

      const store = useClarificationStore()
      store.items = [item]
      store.pendingCount = 1

      await store.snooze(item.id, 24)

      expect(store.items).toHaveLength(0)
      expect(store.pendingCount).toBe(0)
    })
  })

  describe('dismiss', () => {
    it('removes item and decrements pendingCount', async () => {
      const item = makeClarificationItem()
      vi.mocked(clarificationService.dismiss).mockResolvedValue(item)

      const store = useClarificationStore()
      store.items = [item]
      store.pendingCount = 1

      await store.dismiss(item.id)

      expect(store.items).toHaveLength(0)
      expect(store.pendingCount).toBe(0)
    })
  })

  describe('goTo', () => {
    it('updates currentIndex within bounds', () => {
      const store = useClarificationStore()
      store.items = [makeClarificationItem(), makeClarificationItem(), makeClarificationItem()]
      store.goTo(2)
      expect(store.currentIndex).toBe(2)
    })

    it('does not update currentIndex out of bounds', () => {
      const store = useClarificationStore()
      store.items = [makeClarificationItem()]
      store.goTo(5)
      expect(store.currentIndex).toBe(0)
    })
  })

  describe('isEmpty', () => {
    it('true when not loading and no items', () => {
      const store = useClarificationStore()
      store.loading = false
      store.items = []
      expect(store.isEmpty).toBe(true)
    })

    it('false when loading', () => {
      const store = useClarificationStore()
      store.loading = true
      store.items = []
      expect(store.isEmpty).toBe(false)
    })
  })

  describe('progress', () => {
    it('returns current and total', () => {
      const store = useClarificationStore()
      store.items = [makeClarificationItem(), makeClarificationItem()]
      store.currentIndex = 0
      expect(store.progress).toEqual({ current: 1, total: 2 })
    })
  })
})
