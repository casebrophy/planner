import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { taskService } from '@/services/taskService'
import { useToastStore } from './toastStore'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'
import { TaskStatus } from '@/types'

const CACHE_TTL = 5 * 60 * 1000 // 5 minutes

export const useTaskStore = defineStore('task', () => {
  const items = ref<Task[]>([])
  const total = ref(0)
  const page = ref(1)
  const rowsPerPage = ref(20)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastFetchedAt = ref<Record<string, number>>({})
  const filter = ref<TaskFilter>({})
  const orderBy = ref('created_at')
  const currentTask = ref<Task | null>(null)

  const toast = useToastStore()

  const tasksByStatus = computed(() => {
    const groups: Record<string, Task[]> = {}
    for (const task of items.value) {
      const s = task.status
      if (!groups[s]) groups[s] = []
      groups[s]!.push(task)
    }
    return groups
  })

  const hasActiveFilter = computed(() => {
    return !!(filter.value.status || filter.value.priority || filter.value.contextId)
  })

  const overdueCount = computed(() => {
    const now = new Date()
    return items.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < now &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Cancelled,
    ).length
  })

  function cacheKey(): string {
    return JSON.stringify({ filter: filter.value, orderBy: orderBy.value, page: page.value })
  }

  function isCacheValid(): boolean {
    const key = cacheKey()
    const ts = lastFetchedAt.value[key]
    return ts !== undefined && Date.now() - ts < CACHE_TTL
  }

  async function fetchTasks(force = false) {
    if (!force && isCacheValid()) return
    loading.value = true
    error.value = null
    try {
      const result = await taskService.list({
        page: page.value,
        rows: rowsPerPage.value,
        orderBy: orderBy.value,
        filter: filter.value,
      })
      items.value = result.items
      total.value = result.total
      lastFetchedAt.value[cacheKey()] = Date.now()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch tasks'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchTask(id: string) {
    loading.value = true
    error.value = null
    try {
      currentTask.value = await taskService.getById(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch task'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function createTask(task: NewTask) {
    try {
      const created = await taskService.create(task)
      items.value.unshift(created)
      total.value++
      toast.success('Task created')
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to create task'
      toast.error(msg)
      throw e
    }
  }

  async function updateTask(id: string, update: UpdateTask) {
    const idx = items.value.findIndex((t) => t.id === id)
    const backup = idx !== -1 ? { ...items.value[idx]! } : null

    // Optimistic update
    if (idx !== -1) {
      items.value[idx] = { ...items.value[idx]!, ...stripUndefined(update) }
    }
    if (currentTask.value?.id === id) {
      currentTask.value = { ...currentTask.value, ...stripUndefined(update) }
    }

    try {
      const updated = await taskService.update(id, update)
      if (idx !== -1) items.value[idx] = updated
      if (currentTask.value?.id === id) currentTask.value = updated
      toast.success('Task updated')
      return updated
    } catch (e) {
      // Rollback
      if (idx !== -1 && backup) items.value[idx] = backup
      if (currentTask.value?.id === id && backup) currentTask.value = backup
      const msg = e instanceof Error ? e.message : 'Failed to update task'
      toast.error(msg)
      throw e
    }
  }

  async function deleteTask(id: string) {
    const idx = items.value.findIndex((t) => t.id === id)
    const backup = idx !== -1 ? items.value[idx]! : null

    // Optimistic
    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
    }

    try {
      await taskService.delete(id)
      if (currentTask.value?.id === id) currentTask.value = null
      toast.success('Task deleted')
    } catch (e) {
      // Rollback
      if (backup && idx !== -1) {
        items.value.splice(idx, 0, backup)
        total.value++
      }
      const msg = e instanceof Error ? e.message : 'Failed to delete task'
      toast.error(msg)
      throw e
    }
  }

  function setFilter(f: TaskFilter) {
    filter.value = f
    page.value = 1
  }

  function setPage(p: number) {
    page.value = p
  }

  function setOrder(o: string) {
    orderBy.value = o
    page.value = 1
  }

  return {
    items,
    total,
    page,
    rowsPerPage,
    loading,
    error,
    filter,
    orderBy,
    currentTask,
    lastFetchedAt,
    tasksByStatus,
    hasActiveFilter,
    overdueCount,
    fetchTasks,
    fetchTask,
    createTask,
    updateTask,
    deleteTask,
    setFilter,
    setPage,
    setOrder,
  }
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function stripUndefined(obj: any): any {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj)) {
    if (value !== undefined) result[key] = value
  }
  return result
}
