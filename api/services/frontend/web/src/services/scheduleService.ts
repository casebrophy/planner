import { request } from './client'
import type { ScheduleResponse } from '@/types/schedule'

export const scheduleService = {
  async getSchedule(start: string, end: string): Promise<ScheduleResponse> {
    return request<ScheduleResponse>('/api/v1/schedule', {
      params: { start, end },
    })
  },
}
