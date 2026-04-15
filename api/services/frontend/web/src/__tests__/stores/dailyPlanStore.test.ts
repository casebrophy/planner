import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useDailyPlanStore } from '@/stores/dailyPlanStore'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/dailyPlanService', () => ({
  dailyPlanService: {
    getPlan: vi.fn(),
    generate: vi.fn(),
    updateItem: vi.fn(),
    completeItem: vi.fn(),
    dismissItem: vi.fn(),
  },
}))

import { dailyPlanService } from '@/services/dailyPlanService'
import type { DailyPlan, DailyPlanItem } from '@/types/dailyPlan'

function makePlanItem(overrides: Partial<DailyPlanItem> = {}): DailyPlanItem {
  return {
    id: 'item-1',
    planId: 'plan-1',
    taskId: 'task-1',
    taskTitle: 'Test Task',
    position: 0,
    groupName: 'morning',
    groupPosition: 0,
    status: 'pending',
    createdAt: '2026-01-01T08:00:00Z',
    ...overrides,
  }
}

function makePlan(overrides: Partial<DailyPlan> = {}): DailyPlan {
  return {
    id: 'plan-1',
    planDate: '2026-01-01',
    generation: 1,
    modelUsed: 'claude-3',
    createdAt: '2026-01-01T07:00:00Z',
    items: [],
    ...overrides,
  }
}

describe('dailyPlanStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  describe('fetchPlan', () => {
    it('sets loading=true during fetch, then false after', async () => {
      let loadingDuring = false
      vi.mocked(dailyPlanService.getPlan).mockImplementation(async () => {
        loadingDuring = true
        return makePlan()
      })

      const store = useDailyPlanStore()
      expect(store.loading).toBe(false)

      const promise = store.fetchPlan()
      // loading is set synchronously before await
      expect(store.loading).toBe(true)
      await promise
      expect(store.loading).toBe(false)
      expect(loadingDuring).toBe(true)
    })

    it('calls dailyPlanService.getPlan with the given date', async () => {
      vi.mocked(dailyPlanService.getPlan).mockResolvedValue(makePlan())

      const store = useDailyPlanStore()
      await store.fetchPlan('2026-03-15')

      expect(dailyPlanService.getPlan).toHaveBeenCalledWith('2026-03-15')
    })

    it('calls dailyPlanService.getPlan with no argument when date is omitted', async () => {
      vi.mocked(dailyPlanService.getPlan).mockResolvedValue(makePlan())

      const store = useDailyPlanStore()
      await store.fetchPlan()

      expect(dailyPlanService.getPlan).toHaveBeenCalledWith(undefined)
    })

    it('sets plan.value on success', async () => {
      const plan = makePlan({ items: [makePlanItem()] })
      vi.mocked(dailyPlanService.getPlan).mockResolvedValue(plan)

      const store = useDailyPlanStore()
      await store.fetchPlan()

      expect(store.plan).toEqual(plan)
    })

    it('sets loading=false on error', async () => {
      vi.mocked(dailyPlanService.getPlan).mockRejectedValue(new Error('network error'))

      const store = useDailyPlanStore()
      await store.fetchPlan()

      expect(store.loading).toBe(false)
    })

    it('leaves plan unchanged on error', async () => {
      const existing = makePlan()
      vi.mocked(dailyPlanService.getPlan).mockRejectedValueOnce(new Error('fail'))

      const store = useDailyPlanStore()
      store.plan = existing
      await store.fetchPlan()

      expect(store.plan).toEqual(existing)
    })
  })

  describe('regenerate', () => {
    it('sets generating=true during call, then false after', async () => {
      vi.useFakeTimers()
      // Return a plan with items so the polling loop exits immediately after first fetchPlan
      const planWithItems = makePlan({ items: [makePlanItem()] })
      vi.mocked(dailyPlanService.generate).mockResolvedValue({ status: 'accepted' })
      vi.mocked(dailyPlanService.getPlan).mockResolvedValue(planWithItems)

      const store = useDailyPlanStore()
      const promise = store.regenerate()
      expect(store.generating).toBe(true)
      // Advance past the 3s delay so the polling loop can proceed
      await vi.runAllTimersAsync()
      await promise
      expect(store.generating).toBe(false)
      vi.useRealTimers()
    })

    it('calls generate then fetchPlan', async () => {
      vi.useFakeTimers()
      const planWithItems = makePlan({ items: [makePlanItem()] })
      vi.mocked(dailyPlanService.generate).mockResolvedValue({ status: 'accepted' })
      vi.mocked(dailyPlanService.getPlan).mockResolvedValue(planWithItems)

      const store = useDailyPlanStore()
      const regeneratePromise = store.regenerate('2026-03-15')
      await vi.runAllTimersAsync()
      await regeneratePromise

      expect(dailyPlanService.generate).toHaveBeenCalledWith('2026-03-15')
      expect(dailyPlanService.getPlan).toHaveBeenCalledWith('2026-03-15')
      vi.useRealTimers()
    })

    it('sets generating=false on error', async () => {
      vi.mocked(dailyPlanService.generate).mockRejectedValue(new Error('fail'))

      const store = useDailyPlanStore()
      await store.regenerate()

      expect(store.generating).toBe(false)
    })
  })

  describe('completeItem', () => {
    it('sets item status to completed and sets completedAt optimistically', async () => {
      const item = makePlanItem({ id: 'item-1', status: 'pending' })
      const plan = makePlan({ items: [item] })
      vi.mocked(dailyPlanService.completeItem).mockResolvedValue({ ...item, status: 'completed' })

      const store = useDailyPlanStore()
      store.plan = plan
      await store.completeItem('item-1')

      const updated = store.plan!.items.find((i) => i.id === 'item-1')!
      expect(updated.status).toBe('completed')
      expect(updated.completedAt).toBeDefined()
    })

    it('rolls back status and completedAt on error', async () => {
      const item = makePlanItem({ id: 'item-1', status: 'pending', completedAt: undefined })
      const plan = makePlan({ items: [item] })
      vi.mocked(dailyPlanService.completeItem).mockRejectedValue(new Error('fail'))

      const store = useDailyPlanStore()
      store.plan = plan
      await store.completeItem('item-1')

      const rolled = store.plan!.items.find((i) => i.id === 'item-1')!
      expect(rolled.status).toBe('pending')
      expect(rolled.completedAt).toBeUndefined()
    })

    it('does nothing when plan is null', async () => {
      const store = useDailyPlanStore()
      store.plan = null
      await expect(store.completeItem('item-1')).resolves.toBeUndefined()
      expect(dailyPlanService.completeItem).not.toHaveBeenCalled()
    })

    it('does nothing when item is not found in plan', async () => {
      const store = useDailyPlanStore()
      store.plan = makePlan({ items: [makePlanItem({ id: 'item-other' })] })
      await store.completeItem('item-missing')
      expect(dailyPlanService.completeItem).not.toHaveBeenCalled()
    })
  })

  describe('dismissItem', () => {
    it('sets item status to dismissed optimistically with reason and note', async () => {
      const item = makePlanItem({ id: 'item-1', status: 'pending' })
      const plan = makePlan({ items: [item] })
      vi.mocked(dailyPlanService.dismissItem).mockResolvedValue({ ...item, status: 'dismissed' })

      const store = useDailyPlanStore()
      store.plan = plan
      await store.dismissItem('item-1', 'not-relevant', 'skipping today')

      const updated = store.plan!.items.find((i) => i.id === 'item-1')!
      expect(updated.status).toBe('dismissed')
      expect(updated.dismissReason).toBe('not-relevant')
      expect(updated.dismissNote).toBe('skipping today')
    })

    it('calls dailyPlanService.dismissItem with reason and note', async () => {
      const item = makePlanItem({ id: 'item-1' })
      vi.mocked(dailyPlanService.dismissItem).mockResolvedValue({ ...item, status: 'dismissed' })

      const store = useDailyPlanStore()
      store.plan = makePlan({ items: [item] })
      await store.dismissItem('item-1', 'too-long', 'no time')

      expect(dailyPlanService.dismissItem).toHaveBeenCalledWith('item-1', { reason: 'too-long', note: 'no time' })
    })

    it('rolls back status, dismissReason, and dismissNote on error', async () => {
      const item = makePlanItem({
        id: 'item-1',
        status: 'pending',
        dismissReason: undefined,
        dismissNote: undefined,
      })
      const plan = makePlan({ items: [item] })
      vi.mocked(dailyPlanService.dismissItem).mockRejectedValue(new Error('fail'))

      const store = useDailyPlanStore()
      store.plan = plan
      await store.dismissItem('item-1', 'blocked', 'some note')

      const rolled = store.plan!.items.find((i) => i.id === 'item-1')!
      expect(rolled.status).toBe('pending')
      expect(rolled.dismissReason).toBeUndefined()
      expect(rolled.dismissNote).toBeUndefined()
    })

    it('does nothing when plan is null', async () => {
      const store = useDailyPlanStore()
      store.plan = null
      await expect(store.dismissItem('item-1', 'blocked')).resolves.toBeUndefined()
      expect(dailyPlanService.dismissItem).not.toHaveBeenCalled()
    })
  })

  describe('reorderItem', () => {
    it('updates userPosition on the item', async () => {
      const item = makePlanItem({ id: 'item-1', userPosition: 0 })
      const plan = makePlan({ items: [item] })
      vi.mocked(dailyPlanService.updateItem).mockResolvedValue({ ...item, userPosition: 3 })

      const store = useDailyPlanStore()
      store.plan = plan
      await store.reorderItem('item-1', 3)

      const updated = store.plan!.items.find((i) => i.id === 'item-1')!
      expect(updated.userPosition).toBe(3)
    })

    it('calls dailyPlanService.updateItem with the new userPosition', async () => {
      const item = makePlanItem({ id: 'item-1' })
      vi.mocked(dailyPlanService.updateItem).mockResolvedValue({ ...item, userPosition: 5 })

      const store = useDailyPlanStore()
      store.plan = makePlan({ items: [item] })
      await store.reorderItem('item-1', 5)

      expect(dailyPlanService.updateItem).toHaveBeenCalledWith('item-1', { userPosition: 5 })
    })

    it('does nothing when plan is null', async () => {
      const store = useDailyPlanStore()
      store.plan = null
      await expect(store.reorderItem('item-1', 2)).resolves.toBeUndefined()
      expect(dailyPlanService.updateItem).not.toHaveBeenCalled()
    })

    it('does nothing when item is not found in plan', async () => {
      const store = useDailyPlanStore()
      store.plan = makePlan({ items: [makePlanItem({ id: 'item-other' })] })
      await store.reorderItem('item-missing', 2)
      expect(dailyPlanService.updateItem).not.toHaveBeenCalled()
    })
  })
})
