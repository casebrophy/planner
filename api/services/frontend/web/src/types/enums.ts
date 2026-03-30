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

export const ClarificationKind = {
  ContextAssignment: 'context_assignment',
  StaleTask: 'stale_task',
  AmbiguousDeadline: 'ambiguous_deadline',
  NewContext: 'new_context',
  OverlappingContexts: 'overlapping_contexts',
  AmbiguousAction: 'ambiguous_action',
  VoiceReference: 'voice_reference',
  InactivityPrompt: 'inactivity_prompt',
  ContextDebrief: 'context_debrief',
} as const
export type ClarificationKind = (typeof ClarificationKind)[keyof typeof ClarificationKind]

export const ClarificationStatus = {
  Pending: 'pending',
  Snoozed: 'snoozed',
  Resolved: 'resolved',
  Dismissed: 'dismissed',
} as const
export type ClarificationStatus = (typeof ClarificationStatus)[keyof typeof ClarificationStatus]

export const ClarificationKindLabels: Record<ClarificationKind, string> = {
  [ClarificationKind.ContextAssignment]: 'Context Assignment',
  [ClarificationKind.StaleTask]: 'Stale Task',
  [ClarificationKind.AmbiguousDeadline]: 'Ambiguous Deadline',
  [ClarificationKind.NewContext]: 'New Context',
  [ClarificationKind.OverlappingContexts]: 'Overlapping Contexts',
  [ClarificationKind.AmbiguousAction]: 'Ambiguous Action',
  [ClarificationKind.VoiceReference]: 'Voice Reference',
  [ClarificationKind.InactivityPrompt]: 'Inactivity',
  [ClarificationKind.ContextDebrief]: 'Debrief',
}

export const ClarificationKindColors: Record<ClarificationKind, string> = {
  [ClarificationKind.ContextAssignment]: '#f59e0b',
  [ClarificationKind.StaleTask]: '#ef4444',
  [ClarificationKind.AmbiguousDeadline]: '#f97316',
  [ClarificationKind.NewContext]: '#8b5cf6',
  [ClarificationKind.OverlappingContexts]: '#6366f1',
  [ClarificationKind.AmbiguousAction]: '#f59e0b',
  [ClarificationKind.VoiceReference]: '#3b82f6',
  [ClarificationKind.InactivityPrompt]: '#ef4444',
  [ClarificationKind.ContextDebrief]: '#10b981',
}
