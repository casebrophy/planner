import type { ClarificationKind, ClarificationStatus } from './enums'

export interface ClarificationItem {
  id: string
  kind: ClarificationKind
  status: ClarificationStatus
  subjectType: string
  subjectId: string
  question: string
  claudeGuess?: Record<string, unknown>
  reasoning?: string
  answerOptions: Record<string, unknown>
  answer?: Record<string, unknown>
  priorityScore: number
  snoozedUntil?: string
  createdAt: string
  resolvedAt?: string
}

export interface ClarificationCountResponse {
  count: number
}
