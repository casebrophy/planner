export interface DailyPlan {
  id: string
  planDate: string
  generation: number
  modelUsed: string
  createdAt: string
  items: DailyPlanItem[]
}

export interface DailyPlanItem {
  id: string
  planId: string
  taskId: string
  taskTitle: string
  position: number
  groupName: string
  groupPosition: number
  aiDurationMin?: number
  aiPriorityReason?: string
  userPosition?: number
  userDurationMin?: number
  status: string
  dismissReason?: string
  dismissNote?: string
  completedAt?: string
  createdAt: string
}

export interface UpdatePlanItem {
  userPosition?: number
  userDurationMin?: number
}

export interface DismissRequest {
  reason: string
  note?: string
}
