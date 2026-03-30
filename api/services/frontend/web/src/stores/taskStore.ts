import { defineStore } from 'pinia'
import { computed } from 'vue'
import { taskService } from '@/services/taskService'
import { createCRUDStore } from './createCRUDStore'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'
import { TaskStatus } from '@/types'

export const useTaskStore = defineStore('task', () => {
  const crud = createCRUDStore<Task, NewTask, UpdateTask, TaskFilter>({
    name: 'task',
    service: taskService,
    defaultOrderBy: 'created_at',
    defaultRowsPerPage: 20,
  })

  const tasksByStatus = computed(() => {
    const groups: Record<string, Task[]> = {}
    for (const task of crud.items.value) {
      const s = task.status
      if (!groups[s]) groups[s] = []
      groups[s]!.push(task)
    }
    return groups
  })

  const hasActiveFilter = computed(() => {
    const f = crud.filter.value
    return !!(f.status || f.priority || f.contextId)
  })

  const overdueCount = computed(() => {
    const now = new Date()
    return crud.items.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < now &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Cancelled,
    ).length
  })

  return {
    ...crud,
    tasksByStatus,
    hasActiveFilter,
    overdueCount,
  }
})
