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
  const habitGrid = ref<Record<string, string[]>>({})
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

  async function fetchHabitGrid(habitIds: string[], days: number = 30) {
    if (habitIds.length === 0) {
      habitGrid.value = {}
      return
    }

    const to = new Date()
    const from = new Date()
    from.setDate(from.getDate() - days)

    const response = await activityLogService.getBulkLogs(
      'task',
      habitIds,
      from.toISOString(),
      to.toISOString(),
    )

    const grid: Record<string, string[]> = {}
    for (const [subjectId, logs] of Object.entries(response.items)) {
      grid[subjectId] = logs.map(log => log.loggedAt.split('T')[0] ?? log.loggedAt)
    }
    habitGrid.value = grid
  }

  async function logActivity(subjectType: string, subjectId: string, value?: string) {
    const entry = await crud.create({ subjectType, subjectId, value })
    await fetchStreaks(subjectType, subjectId)
    return entry
  }

  return {
    ...crud,
    streaks,
    habitGrid,
    fetchStreaks,
    fetchHabitGrid,
    logActivity,
  }
})
