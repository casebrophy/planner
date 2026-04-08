import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useContextStore } from '@/stores/contextStore'
import { makeContext } from '../helpers/testFactories'
import { ContextStatus } from '@/types'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/contextService', () => ({
  contextService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
})

describe('contextStore — domain extensions', () => {
  describe('contextsByStatus', () => {
    it('groups contexts into Active, Paused, and Closed buckets', () => {
      const store = useContextStore()
      const active1 = makeContext({ status: ContextStatus.Active })
      const active2 = makeContext({ status: ContextStatus.Active })
      const paused = makeContext({ status: ContextStatus.Paused })
      const closed = makeContext({ status: ContextStatus.Closed })

      store.items = [active1, active2, paused, closed]

      const groups = store.contextsByStatus
      expect(groups[ContextStatus.Active]).toEqual([active1, active2])
      expect(groups[ContextStatus.Paused]).toEqual([paused])
      expect(groups[ContextStatus.Closed]).toEqual([closed])
    })

    it('returns empty arrays for buckets with no matching contexts', () => {
      const store = useContextStore()
      store.items = [makeContext({ status: ContextStatus.Active })]

      const groups = store.contextsByStatus
      expect(groups[ContextStatus.Paused]).toEqual([])
      expect(groups[ContextStatus.Closed]).toEqual([])
    })
  })

  describe('activeCount / pausedCount / closedCount', () => {
    it('returns correct counts per status', () => {
      const store = useContextStore()
      store.items = [
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Active }),
        makeContext({ status: ContextStatus.Paused }),
        makeContext({ status: ContextStatus.Closed }),
        makeContext({ status: ContextStatus.Closed }),
        makeContext({ status: ContextStatus.Closed }),
      ]

      expect(store.activeCount).toBe(2)
      expect(store.pausedCount).toBe(1)
      expect(store.closedCount).toBe(3)
    })

    it('returns zero counts when items is empty', () => {
      const store = useContextStore()
      store.items = []

      expect(store.activeCount).toBe(0)
      expect(store.pausedCount).toBe(0)
      expect(store.closedCount).toBe(0)
    })
  })
})
