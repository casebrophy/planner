import { request } from './client'
import { createCRUDService } from './createCRUDService'
import type { ActivityLog, NewActivityLog, ActivityLogFilter, StreakInfo } from '@/types'

const crud = createCRUDService<ActivityLog, NewActivityLog, never, ActivityLogFilter>({
  basePath: '/api/v1/activity-logs',
  mapFilter: (f) => ({
    subject_type: f.subjectType,
    subject_id: f.subjectId,
    start_date: f.startDate,
    end_date: f.endDate,
  }),
})

export const activityLogService = {
  ...crud,

  async getStreaks(subjectType: string, subjectId: string): Promise<StreakInfo> {
    return request<StreakInfo>(`/api/v1/activity-logs/streaks/${subjectType}/${subjectId}`)
  },
}
