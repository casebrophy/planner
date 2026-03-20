import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useContextStore } from '@/stores/contextStore'
import { contextService } from '@/services/contextService'
import { makeContext } from '../helpers/testFactories'
import { ContextStatus } from '@/types'
import type { ContextEvent, NewEvent } from '@/types'

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
    listEvents: vi.fn(),
    addEvent: vi.fn(),
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

  describe('fetchEvents', () => {
    it('loads events and total from the service', async () => {
      const store = useContextStore()
      const event1: ContextEvent = {
        id: 'ev-1',
        contextId: 'ctx-1',
        kind: 'note',
        content: 'First event',
        createdAt: new Date().toISOString(),
      }
      const event2: ContextEvent = {
        id: 'ev-2',
        contextId: 'ctx-1',
        kind: 'note',
        content: 'Second event',
        createdAt: new Date().toISOString(),
      }
      vi.mocked(contextService.listEvents).mockResolvedValue({ items: [event1, event2], total: 2, page: 1, rowsPerPage: 50 })

      await store.fetchEvents('ctx-1')

      expect(contextService.listEvents).toHaveBeenCalledWith('ctx-1', { page: 1, rows: 50 })
      expect(store.events).toEqual([event1, event2])
      expect(store.eventsTotal).toBe(2)
    })

    it('passes the page argument to the service', async () => {
      const store = useContextStore()
      vi.mocked(contextService.listEvents).mockResolvedValue({ items: [], total: 0, page: 3, rowsPerPage: 50 })

      await store.fetchEvents('ctx-2', 3)

      expect(contextService.listEvents).toHaveBeenCalledWith('ctx-2', { page: 3, rows: 50 })
    })
  })

  describe('addEvent', () => {
    it('prepends the new event to the events array and increments total', async () => {
      const store = useContextStore()
      const existing: ContextEvent = {
        id: 'ev-old',
        contextId: 'ctx-1',
        kind: 'note',
        content: 'Older event',
        createdAt: new Date().toISOString(),
      }
      store.events = [existing]
      store.eventsTotal = 1

      const created: ContextEvent = {
        id: 'ev-new',
        contextId: 'ctx-1',
        kind: 'note',
        content: 'Brand new event',
        createdAt: new Date().toISOString(),
      }
      vi.mocked(contextService.addEvent).mockResolvedValue(created)

      const newEvent: NewEvent = { kind: 'note', content: 'Brand new event' }
      const result = await store.addEvent('ctx-1', newEvent)

      expect(contextService.addEvent).toHaveBeenCalledWith('ctx-1', newEvent)
      expect(result).toEqual(created)
      expect(store.events[0]).toEqual(created)
      expect(store.events[1]).toEqual(existing)
      expect(store.eventsTotal).toBe(2)
    })

    it('re-throws on service failure and does not mutate events', async () => {
      const store = useContextStore()
      store.events = []
      store.eventsTotal = 0

      vi.mocked(contextService.addEvent).mockRejectedValue(new Error('Server error'))

      const newEvent: NewEvent = { kind: 'note', content: 'Will fail' }
      await expect(store.addEvent('ctx-1', newEvent)).rejects.toThrow('Server error')

      expect(store.events).toEqual([])
      expect(store.eventsTotal).toBe(0)
    })
  })
})
