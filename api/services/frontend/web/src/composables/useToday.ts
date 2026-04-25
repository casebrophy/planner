import { onMounted, computed, ref } from 'vue'
import { useContextStore } from '@/stores/contextStore'
import { usePolling } from './usePolling'
import { TaskStatus, TaskPriority, TaskEnergy } from '@/types'
import { fetchAllPages } from '@/services/fetchAllPages'
import { taskService } from '@/services/taskService'
import type { Task } from '@/types'

export function useToday() {
  const contextStore = useContextStore()
  const tasks = ref<Task[]>([])
  const contexts = ref<any[]>([])
  const loading = ref(false)

  function startOfToday(): Date {
    const d = new Date()
    d.setHours(0, 0, 0, 0)
    return d
  }

  function endOfToday(): Date {
    const d = new Date()
    d.setHours(23, 59, 59, 999)
    return d
  }

  const overdueTasks = computed(() =>
    tasks.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < startOfToday() &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Dismissed,
    ),
  )

  const dueTodayTasks = computed(() => {
    const start = startOfToday()
    const end = endOfToday()
    return tasks.value.filter((t) => {
      if (!t.dueDate) return false
      const due = new Date(t.dueDate)
      return due >= start && due <= end
    })
  })

  const blockedTasks = computed(() =>
    tasks.value.filter((t) => t.status === TaskStatus.Blocked),
  )

  const unschedulableTasks = computed(() =>
    tasks.value.filter(
      (t) =>
        t.status === TaskStatus.Open &&
        t.priority === TaskPriority.Medium &&
        t.energy === TaskEnergy.Medium &&
        !t.dueDate &&
        !t.contextId,
    ),
  )

  const contextMap = computed(() => {
    const map: Record<string, string> = {}
    for (const ctx of contexts.value) {
      map[ctx.id] = ctx.title
    }
    return map
  })

  const counts = computed(() => ({
    overdue: overdueTasks.value.length,
    dueToday: dueTodayTasks.value.length,
    blocked: blockedTasks.value.length,
    unschedulable: unschedulableTasks.value.length,
  }))

  async function load() {
    loading.value = true
    try {
      const tasksResult = await fetchAllPages(taskService, {
        filter: { excludeStatuses: [TaskStatus.Done, TaskStatus.Dismissed] },
      })
      tasks.value = tasksResult.items
      await contextStore.fetchAll()
      contexts.value = contextStore.items
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  usePolling(load)

  return {
    loading,
    overdueTasks,
    dueTodayTasks,
    blockedTasks,
    unschedulableTasks,
    contextMap,
    counts,
    refresh: load,
  }
}
