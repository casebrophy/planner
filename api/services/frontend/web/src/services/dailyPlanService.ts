import { request } from './client'
import type { DailyPlan, DailyPlanItem, UpdatePlanItem, DismissRequest } from '@/types/dailyPlan'

export const dailyPlanService = {
  async getPlan(date?: string): Promise<DailyPlan> {
    const params = date ? `?date=${date}` : ''
    return request<DailyPlan>(`/api/v1/daily-plan${params}`)
  },

  async generate(date?: string): Promise<DailyPlan> {
    const params = date ? `?date=${date}` : ''
    return request<DailyPlan>(`/api/v1/daily-plan/generate${params}`, { method: 'POST' })
  },

  async updateItem(itemId: string, data: UpdatePlanItem): Promise<DailyPlanItem> {
    return request<DailyPlanItem>(`/api/v1/daily-plan/items/${itemId}`, { method: 'PUT', body: data })
  },

  async completeItem(itemId: string): Promise<DailyPlanItem> {
    return request<DailyPlanItem>(`/api/v1/daily-plan/items/${itemId}/complete`, { method: 'POST' })
  },

  async dismissItem(itemId: string, data: DismissRequest): Promise<DailyPlanItem> {
    return request<DailyPlanItem>(`/api/v1/daily-plan/items/${itemId}/dismiss`, { method: 'POST', body: data })
  },
}
