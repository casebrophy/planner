import { defineStore } from 'pinia'
import { ref } from 'vue'
import { activityLogService } from '@/services/activityLogService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { ActivityLog, NewActivityLog, ActivityLogFilter, StreakInfo } from '@/types'

/** Map of habitId -> { 'YYYY-MM-DD': boolean } */
export type HabitGridMap = Record<string, Record<string, boolean>>

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

    // Convert logs to date strings grouped by subject ID
    const grid: Record<string, string[]> = {}
    for (const [subjectId, logs] of Object.entries(response.items)) {
      grid[subjectId] = logs.map(log => log.loggedAt.split('T')[0] ?? log.loggedAt)
    }
    habitGrid.value = grid
  }

  async function logActivity(subjectType: string, subjectId: string, value?: string) {
    const entry = await crud.create({ subjectType, subjectId, value })
    // Refresh streaks after logging
    await fetchStreaks(subjectType, subjectId)
    return entry
  }

  const habitGrid = ref<HabitGridMap>({})

  async function fetchHabitGrid(habitIds: string[], days: number) {
    // Build date range
    const now = new Date()
    const start = new Date(now)
    start.setDate(start.getDate() - days)
    const startDate = start.toISOString().slice(0, 10)
    const endDate = now.toISOString().slice(0, 10)

    const grid: HabitGridMap = {}
    for (const id of habitIds) {
      grid[id] = {}
    }

    try {
      // Fetch activity logs for each habit in the date range
      for (const id of habitIds) {
        const result = await activityLogService.list({
          page: 1,
          rows: 500,
          orderBy: 'logged_at',
          filter: {
            subjectType: 'task',
            subjectId: id,
            startDate,
            endDate,
          },
        })
        const dayMap: Record<string, boolean> = {}
        for (const log of result.items) {
          const key = log.loggedAt.slice(0, 10)
          dayMap[key] = true
        }
        grid[id] = dayMap
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch habit grid'
      toast.error(msg)
    }

    habitGrid.value = grid
  }

  return {
    ...crud,
    streaks,
    habitGrid,
    fetchStreaks,
    fetchHabitGrid,
    logActivity,
    habitGrid,
    fetchHabitGrid,
  }
})
