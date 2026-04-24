import { onMounted, computed, ref } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useContextStore } from '@/stores/contextStore'
import { storeToRefs } from 'pinia'
import { usePolling } from './usePolling'
import { TaskStatus, ContextStatus } from '@/types'
import { activityLogService } from '@/services/activityLogService'
import { useToastStore } from '@/stores/toastStore'
import type { ActivityLog } from '@/types'

export interface WeekBucket {
  weekLabel: string
  completed: number
}

export interface ContextBacklog {
  contextId: string
  contextTitle: string
  openCount: number
  blockedCount: number
  total: number
}

export function useDashboard() {
  const taskStore = useTaskStore()
  const contextStore = useContextStore()
  const { items: tasks } = storeToRefs(taskStore)
  const { items: contexts } = storeToRefs(contextStore)
  const loading = ref(false)
  const toasts = useToastStore()

  // Activity logs for the last 4 weeks
  const activityLogs = ref<ActivityLog[]>([])

  // --- Completion trend ---
  const completionTrend = computed<WeekBucket[]>(() => {
    const now = new Date()
    const buckets: WeekBucket[] = []

    for (let i = 3; i >= 0; i--) {
      const weekStart = new Date(now)
      weekStart.setDate(now.getDate() - now.getDay() - i * 7)
      weekStart.setHours(0, 0, 0, 0)

      const weekEnd = new Date(weekStart)
      weekEnd.setDate(weekStart.getDate() + 6)
      weekEnd.setHours(23, 59, 59, 999)

      const completed = activityLogs.value.filter((log) => {
        if (log.subjectType !== 'task') return false
        const loggedAt = new Date(log.loggedAt)
        return loggedAt >= weekStart && loggedAt <= weekEnd
      }).length

      // Label: "Apr 1" style
      const label = weekStart.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
      buckets.push({ weekLabel: label, completed })
    }

    return buckets
  })

  // --- Growing backlogs ---
  const growingBacklogs = computed<ContextBacklog[]>(() => {
    const contextMap: Record<string, ContextBacklog> = {}

    for (const task of tasks.value) {
      if (task.status !== TaskStatus.Open && task.status !== TaskStatus.Blocked) continue
      const ctxId = task.contextId ?? '__none__'
      if (!contextMap[ctxId]) {
        const ctx = contexts.value.find((c) => c.id === ctxId)
        contextMap[ctxId] = {
          contextId: ctxId,
          contextTitle: ctx?.title ?? 'No Context',
          openCount: 0,
          blockedCount: 0,
          total: 0,
        }
      }
      if (task.status === TaskStatus.Open) contextMap[ctxId].openCount++
      if (task.status === TaskStatus.Blocked) contextMap[ctxId].blockedCount++
      contextMap[ctxId].total++
    }

    return Object.values(contextMap)
      .sort((a, b) => b.total - a.total)
      .slice(0, 5)
  })

  // --- Repeatedly dismissed tasks ---
  const repeatedlyDismissed = computed(() =>
    tasks.value
      .filter((t) => t.status === TaskStatus.Dismissed)
      .sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
      .slice(0, 10),
  )

  // --- Inactive contexts ---
  const inactiveContexts = computed(() => {
    const cutoff = new Date()
    cutoff.setDate(cutoff.getDate() - 14)

    const activeContextIds = new Set(
      activityLogs.value
        .filter((log) => {
          if (log.subjectType !== 'context') return false
          return new Date(log.loggedAt) >= cutoff
        })
        .map((log) => log.subjectId),
    )

    // Also consider tasks updated recently as context activity
    const recentlyActiveContextIds = new Set(
      tasks.value
        .filter((t) => new Date(t.updatedAt) >= cutoff)
        .map((t) => t.contextId)
        .filter(Boolean) as string[],
    )

    return contexts.value.filter(
      (ctx) =>
        ctx.status === ContextStatus.Active &&
        !activeContextIds.has(ctx.id) &&
        !recentlyActiveContextIds.has(ctx.id),
    )
  })

  async function load() {
    loading.value = true
    try {
      // Fetch tasks and contexts
      await Promise.all([taskStore.fetchList(true), contextStore.fetchList(true)])

      // Fetch activity logs for the last 4 weeks
      const fourWeeksAgo = new Date()
      fourWeeksAgo.setDate(fourWeeksAgo.getDate() - 28)
      fourWeeksAgo.setHours(0, 0, 0, 0)

      const now = new Date()

      const result = await activityLogService.list({
        rows: 100,
        filter: {
          subjectType: 'task',
          startDate: fourWeeksAgo.toISOString(),
          endDate: now.toISOString(),
        },
      })
      activityLogs.value = result.items ?? []
    } catch (e) {
      console.error('dashboard load failed', e)
      activityLogs.value = []
      toasts.error(e instanceof Error ? e.message : 'Failed to load dashboard data')
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  usePolling(load)

  return {
    loading,
    completionTrend,
    growingBacklogs,
    repeatedlyDismissed,
    inactiveContexts,
    refresh: load,
  }
}
