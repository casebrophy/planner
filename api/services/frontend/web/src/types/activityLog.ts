export interface ActivityLog {
  id: string
  subjectType: string
  subjectId: string
  value?: string
  loggedAt: string
}

export interface NewActivityLog {
  subjectType: string
  subjectId: string
  value?: string
}

export interface ActivityLogFilter {
  subjectType?: string
  subjectId?: string
  startDate?: string
  endDate?: string
}

export interface StreakInfo {
  current: number
  longest: number
  totalCount: number
  lastLogged?: string
}

export interface BulkLogsResponse {
  items: Record<string, ActivityLog[]>
}
