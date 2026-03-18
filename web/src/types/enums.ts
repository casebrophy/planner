export const TaskStatus = {
  Todo: 'todo',
  InProgress: 'in_progress',
  Done: 'done',
  Cancelled: 'cancelled',
} as const
export type TaskStatus = (typeof TaskStatus)[keyof typeof TaskStatus]

export const TaskPriority = {
  Low: 'low',
  Medium: 'medium',
  High: 'high',
  Urgent: 'urgent',
} as const
export type TaskPriority = (typeof TaskPriority)[keyof typeof TaskPriority]

export const TaskEnergy = {
  Low: 'low',
  Medium: 'medium',
  High: 'high',
} as const
export type TaskEnergy = (typeof TaskEnergy)[keyof typeof TaskEnergy]

export const ContextStatus = {
  Active: 'active',
  Paused: 'paused',
  Closed: 'closed',
} as const
export type ContextStatus = (typeof ContextStatus)[keyof typeof ContextStatus]

export const TaskStatusLabels: Record<TaskStatus, string> = {
  [TaskStatus.Todo]: 'To Do',
  [TaskStatus.InProgress]: 'In Progress',
  [TaskStatus.Done]: 'Done',
  [TaskStatus.Cancelled]: 'Cancelled',
}

export const TaskPriorityLabels: Record<TaskPriority, string> = {
  [TaskPriority.Low]: 'Low',
  [TaskPriority.Medium]: 'Medium',
  [TaskPriority.High]: 'High',
  [TaskPriority.Urgent]: 'Urgent',
}

export const TaskEnergyLabels: Record<TaskEnergy, string> = {
  [TaskEnergy.Low]: 'Low',
  [TaskEnergy.Medium]: 'Medium',
  [TaskEnergy.High]: 'High',
}

export const ContextStatusLabels: Record<ContextStatus, string> = {
  [ContextStatus.Active]: 'Active',
  [ContextStatus.Paused]: 'Paused',
  [ContextStatus.Closed]: 'Closed',
}

export const StatusColors: Record<string, string> = {
  todo: '#6b7280',
  in_progress: '#3b82f6',
  done: '#22c55e',
  cancelled: '#ef4444',
  active: '#22c55e',
  paused: '#eab308',
  closed: '#6b7280',
}

export const PriorityColors: Record<TaskPriority, string> = {
  [TaskPriority.Low]: '#6b7280',
  [TaskPriority.Medium]: '#3b82f6',
  [TaskPriority.High]: '#f97316',
  [TaskPriority.Urgent]: '#ef4444',
}
