import { defineStore } from 'pinia'
import { ref } from 'vue'
import { activityLogService } from '@/services/activityLogService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { ActivityLog, NewActivityLog, ActivityLogFilter, StreakInfo } from '@/types'

export const useActivityLogStore = defineStore('activityLog', () => {
  const crud = createCRUDStore<ActivityLog, NewActivityLog, never, ActivityLogFilter>({
    name: 'activity log',
    service: activityLogService,
    defaultOrderBy: 'logged_at',
    defaultRowsPerPage: 50,
  })

  const streaks = ref<Record<string, StreakInfo>>({})
  const toast = useToastStore()

  async function fetchStreaks(subjectType: string, subjectId: string) {
    const key = `${subjectType}:${subjectId}`
    try {
      streaks.value[key] = await activityLogService.getStreaks(subjectType, subjectId)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch streaks'
      toast.error(msg)
    }
  }

  async function logActivity(subjectType: string, subjectId: string, value?: string) {
    const entry = await crud.create({ subjectType, subjectId, value })
    // Refresh streaks after logging
    await fetchStreaks(subjectType, subjectId)
    return entry
  }

  return {
    ...crud,
    streaks,
    fetchStreaks,
    logActivity,
  }
})
