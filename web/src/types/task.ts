import type { TaskStatus, TaskPriority, TaskEnergy } from './enums'

export interface Task {
  id: string
  contextId?: string
  title: string
  description: string
  status: TaskStatus
  priority: TaskPriority
  energy: TaskEnergy
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
  createdAt: string
  updatedAt: string
  completedAt?: string
}

export interface NewTask {
  title: string
  description: string
  contextId?: string
  priority: TaskPriority
  energy: TaskEnergy
  durationMin?: number
  dueDate?: string
}

export interface UpdateTask {
  title?: string
  description?: string
  contextId?: string
  status?: TaskStatus
  priority?: TaskPriority
  energy?: TaskEnergy
  durationMin?: number
  dueDate?: string
  scheduledAt?: string
}

export interface TaskFilter {
  status?: TaskStatus
  priority?: TaskPriority
  contextId?: string
  startDueDate?: string
  endDueDate?: string
}
