export type HabitGridEntry = { id: string; date: string }
export type HabitGridMap = Record<string, HabitGridEntry[]>

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
  const habitGrid = ref<HabitGridMap>({})
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

    const grid: HabitGridMap = {}
    for (const [subjectId, logs] of Object.entries(response.items)) {
      grid[subjectId] = logs.map(log => ({
        id: log.id,
        date: log.loggedAt.split('T')[0] ?? log.loggedAt,
      }))
    }
    habitGrid.value = grid
  }

  async function toggleHabitDay(habitId: string, day: Date) {
    const dateStr = day.toISOString().slice(0, 10)
    const row = habitGrid.value[habitId] ?? []
    const existing = row.find(e => e.date === dateStr)

    if (existing) {
      // Remove the log
      await crud.remove(existing.id)
      habitGrid.value[habitId] = row.filter(e => e.id !== existing.id)
    } else {
      // Create a new log
      const log = await crud.create({ subjectType: 'task', subjectId: habitId })
      if (log) {
        const entry: HabitGridEntry = {
          id: log.id,
          date: log.loggedAt.split('T')[0] ?? log.loggedAt,
        }
        habitGrid.value[habitId] = [...row, entry]
      }
    }
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
    toggleHabitDay,
    logActivity,
  }
})
