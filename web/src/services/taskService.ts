import { request } from './client'
import type { Task, NewTask, UpdateTask, TaskFilter, QueryResult, ListParams } from '@/types'

interface TaskListParams extends ListParams {
  filter?: TaskFilter
}

interface TaskQueryResponse {
  items: Task[]
  total: number
  page: number
  rowsPerPage: number
}

export const taskService = {
  async list(params: TaskListParams = {}): Promise<QueryResult<Task>> {
    const queryParams: Record<string, string | number | undefined> = {
      page: params.page,
      rows: params.rows,
      orderBy: params.orderBy,
      status: params.filter?.status,
      priority: params.filter?.priority,
      context_id: params.filter?.contextId,
      start_due_date: params.filter?.startDueDate,
      end_due_date: params.filter?.endDueDate,
    }

    return request<TaskQueryResponse>('/api/v1/tasks', { params: queryParams })
  },

  async getById(id: string): Promise<Task> {
    return request<Task>(`/api/v1/tasks/${id}`)
  },

  async create(task: NewTask): Promise<Task> {
    return request<Task>('/api/v1/tasks', { method: 'POST', body: task })
  },

  async update(id: string, task: UpdateTask): Promise<Task> {
    return request<Task>(`/api/v1/tasks/${id}`, { method: 'PUT', body: task })
  },

  async delete(id: string): Promise<void> {
    return request<void>(`/api/v1/tasks/${id}`, { method: 'DELETE' })
  },
}
