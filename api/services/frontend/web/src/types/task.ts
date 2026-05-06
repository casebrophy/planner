import type { TaskStatus, TaskPriority } from './enums'

export interface Task {
  id: string
  contextId?: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  energy: 'low' | 'medium' | 'high'
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
  recurrenceParentId?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
  trackOutcome?: boolean
  unconfirmed?: boolean
}

export interface NewTask {
  title: string
  description: string
  contextId?: string
  priority: TaskPriority
  durationMin?: number
  dueDate?: string
  recurrenceRule?: string
}

export interface UpdateTask {
  title?: string
  description?: string
  contextId?: string
  status?: TaskStatus
  priority?: TaskPriority
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  blockedReason?: string
  recurrenceRule?: string
  trackOutcome?: boolean
}

export interface TaskFilter {
  status?: TaskStatus
  excludeStatuses?: TaskStatus[]
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
  hasRecurrence?: boolean
}
