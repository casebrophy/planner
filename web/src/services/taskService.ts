import { createCRUDService } from './createCRUDService'
import type { Task, NewTask, UpdateTask, TaskFilter } from '@/types'

export const taskService = createCRUDService<Task, NewTask, UpdateTask, TaskFilter>({
  basePath: '/api/v1/tasks',
  mapFilter: (f) => ({
    status: f.status,
    priority: f.priority,
    context_id: f.contextId,
    start_due_date: f.startDueDate,
    end_due_date: f.endDueDate,
  }),
})
