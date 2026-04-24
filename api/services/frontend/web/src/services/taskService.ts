import { createCRUDService } from './createCRUDService'
import { request } from './client'
import type { Task, NewTask, UpdateTask, TaskFilter, Note } from '@/types'

const taskCrud = createCRUDService<Task, NewTask, UpdateTask, TaskFilter>({
  basePath: '/api/v1/tasks',
  mapFilter: (f) => ({
    status: f.status,
    exclude_status: f.excludeStatuses?.length ? f.excludeStatuses.join(',') : undefined,
    priority: f.priority,
    context_id: f.contextId,
    start_due_date: f.startDueDate,
    end_due_date: f.endDueDate,
    has_recurrence: f.hasRecurrence?.toString(),
  }),
})

export const taskService = {
  ...taskCrud,
  deleteBatch: (ids: string[]): Promise<void> =>
    request<void>('/api/v1/tasks/batch', { method: 'DELETE', body: { ids } }),
  convertTaskToNote: (taskId: string): Promise<Note> =>
    request<Note>(`/api/v1/tasks/${taskId}/convert-to-note`, { method: 'POST' }),
}
