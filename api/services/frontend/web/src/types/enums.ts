import type { ClarificationKindValue } from './generated/clarification-kind'

export const TaskStatus = {
  Open: 'open',
  Blocked: 'blocked',
  Done: 'done',
  Dismissed: 'dismissed',
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
  [TaskStatus.Open]: 'Open',
  [TaskStatus.Blocked]: 'Blocked',
  [TaskStatus.Done]: 'Done',
  [TaskStatus.Dismissed]: 'Dismissed',
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

export const ContextKind = {
  Project: 'project',
  Area: 'area',
} as const
export type ContextKind = (typeof ContextKind)[keyof typeof ContextKind]

export const ContextKindLabels: Record<ContextKind, string> = {
  [ContextKind.Project]: 'Project',
  [ContextKind.Area]: 'Area',
}

export const ContextKindColors: Record<ContextKind, string> = {
  [ContextKind.Project]: '#3b82f6',
  [ContextKind.Area]: '#8b5cf6',
}

export const ContextStatusLabels: Record<ContextStatus, string> = {
  [ContextStatus.Active]: 'Active',
  [ContextStatus.Paused]: 'Paused',
  [ContextStatus.Closed]: 'Closed',
}

export const StatusColors: Record<string, string> = {
  open: '#6b7280',
  blocked: '#f59e0b',
  done: '#22c55e',
  dismissed: '#9ca3af',
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
  TaskDebrief: 'task_debrief',
} as const satisfies Record<string, ClarificationKindValue>

// ClarificationKind is the authoritative TypeScript type — derived from the generated union.
// If Go adds a new kind, ClarificationKindLabels and ClarificationKindColors below will
// produce a TypeScript error until they are updated.
export type ClarificationKind = ClarificationKindValue

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
  [ClarificationKind.TaskDebrief]: 'Task Debrief',
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
  [ClarificationKind.TaskDebrief]: '#06b6d4',
}
