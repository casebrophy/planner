import { describe, it, expect, beforeEach } from 'vitest'
import { dailyPlanService } from '@/services/dailyPlanService'
import { setupMockFetch } from '../helpers/mockFetch'
import type { DailyPlan, DailyPlanItem } from '@/types/dailyPlan'

const { mockFetch, jsonResponse } = setupMockFetch()

beforeEach(() => {
  mockFetch.mockReset()
})

function makePlanItem(overrides: Partial<DailyPlanItem> = {}): DailyPlanItem {
  return {
    id: 'item-1',
    planId: 'plan-1',
    taskId: 'task-1',
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

describe('dailyPlanService', () => {
  describe('getPlan', () => {
    it('calls GET /api/v1/daily-plan with no date param when date is omitted', async () => {
      mockFetch.mockReturnValue(jsonResponse(makePlan()))

      await dailyPlanService.getPlan()

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('/api/v1/daily-plan')
      expect(url).not.toContain('date=')
      expect(mockFetch.mock.calls[0]![1]?.method ?? 'GET').toBe('GET')
    })

    it('appends date query param when date is provided', async () => {
      mockFetch.mockReturnValue(jsonResponse(makePlan({ planDate: '2026-03-15' })))

      await dailyPlanService.getPlan('2026-03-15')

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('date=2026-03-15')
    })

    it('returns the parsed DailyPlan from the response', async () => {
      const plan = makePlan({ id: 'plan-abc', items: [makePlanItem()] })
      mockFetch.mockReturnValue(jsonResponse(plan))

      const result = await dailyPlanService.getPlan()

      expect(result.id).toBe('plan-abc')
      expect(result.items).toHaveLength(1)
    })
  })

  describe('generate', () => {
    it('calls POST /api/v1/daily-plan/generate with no date param when omitted', async () => {
      mockFetch.mockReturnValue(jsonResponse({ status: 'accepted' }))

      await dailyPlanService.generate()

      const url = mockFetch.mock.calls[0]![0] as string
      const opts = mockFetch.mock.calls[0]![1]
      expect(url).toContain('/api/v1/daily-plan/generate')
      expect(url).not.toContain('date=')
      expect(opts?.method).toBe('POST')
    })

    it('appends date query param when date is provided', async () => {
      mockFetch.mockReturnValue(jsonResponse({ status: 'accepted' }))

      await dailyPlanService.generate('2026-03-15')

      const url = mockFetch.mock.calls[0]![0] as string
      expect(url).toContain('date=2026-03-15')
    })

    it('returns the GenerateAccepted response', async () => {
      mockFetch.mockReturnValue(jsonResponse({ status: 'accepted' }))

      const result = await dailyPlanService.generate()

      expect(result.status).toBe('accepted')
    })
  })

  describe('updateItem', () => {
    it('calls PUT /api/v1/daily-plan/items/:id', async () => {
      const item = makePlanItem({ id: 'item-42', userPosition: 3 })
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.updateItem('item-42', { userPosition: 3 })

      const url = mockFetch.mock.calls[0]![0] as string
      const opts = mockFetch.mock.calls[0]![1]
      expect(url).toContain('/api/v1/daily-plan/items/item-42')
      expect(opts?.method).toBe('PUT')
    })

    it('sends the update payload as JSON body', async () => {
      const item = makePlanItem()
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.updateItem('item-1', { userPosition: 5, userDurationMin: 30 })

      const opts = mockFetch.mock.calls[0]![1]
      const body = JSON.parse(opts?.body as string)
      expect(body.userPosition).toBe(5)
      expect(body.userDurationMin).toBe(30)
    })

    it('returns the updated DailyPlanItem', async () => {
      const item = makePlanItem({ userPosition: 7 })
      mockFetch.mockReturnValue(jsonResponse(item))

      const result = await dailyPlanService.updateItem('item-1', { userPosition: 7 })

      expect(result.userPosition).toBe(7)
    })
  })

  describe('completeItem', () => {
    it('calls POST /api/v1/daily-plan/items/:id/complete', async () => {
      const item = makePlanItem({ status: 'completed' })
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.completeItem('item-99')

      const url = mockFetch.mock.calls[0]![0] as string
      const opts = mockFetch.mock.calls[0]![1]
      expect(url).toContain('/api/v1/daily-plan/items/item-99/complete')
      expect(opts?.method).toBe('POST')
    })

    it('returns the updated DailyPlanItem with completed status', async () => {
      const item = makePlanItem({ status: 'completed', completedAt: '2026-01-01T09:00:00Z' })
      mockFetch.mockReturnValue(jsonResponse(item))

      const result = await dailyPlanService.completeItem('item-1')

      expect(result.status).toBe('completed')
      expect(result.completedAt).toBe('2026-01-01T09:00:00Z')
    })
  })

  describe('dismissItem', () => {
    it('calls POST /api/v1/daily-plan/items/:id/dismiss', async () => {
      const item = makePlanItem({ status: 'dismissed' })
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.dismissItem('item-77', { reason: 'not-relevant' })

      const url = mockFetch.mock.calls[0]![0] as string
      const opts = mockFetch.mock.calls[0]![1]
      expect(url).toContain('/api/v1/daily-plan/items/item-77/dismiss')
      expect(opts?.method).toBe('POST')
    })

    it('sends reason and note in the request body', async () => {
      const item = makePlanItem({ status: 'dismissed' })
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.dismissItem('item-1', { reason: 'too-long', note: 'deprioritized' })

      const opts = mockFetch.mock.calls[0]![1]
      const body = JSON.parse(opts?.body as string)
      expect(body.reason).toBe('too-long')
      expect(body.note).toBe('deprioritized')
    })

    it('sends only reason when note is omitted', async () => {
      const item = makePlanItem({ status: 'dismissed' })
      mockFetch.mockReturnValue(jsonResponse(item))

      await dailyPlanService.dismissItem('item-1', { reason: 'blocked' })

      const opts = mockFetch.mock.calls[0]![1]
      const body = JSON.parse(opts?.body as string)
      expect(body.reason).toBe('blocked')
      expect(body.note).toBeUndefined()
    })

    it('returns the updated DailyPlanItem with dismissed status', async () => {
      const item = makePlanItem({ status: 'dismissed', dismissReason: 'not-relevant' })
      mockFetch.mockReturnValue(jsonResponse(item))

      const result = await dailyPlanService.dismissItem('item-1', { reason: 'not-relevant' })

      expect(result.status).toBe('dismissed')
      expect(result.dismissReason).toBe('not-relevant')
    })
  })
})
