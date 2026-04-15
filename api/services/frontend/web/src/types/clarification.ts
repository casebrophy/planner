import type { ClarificationKind, ClarificationStatus } from './enums'
import type {
  ContextAssignmentOptions,
  NewContextOptions,
  AmbiguousActionOptions,
  AmbiguousDeadlineOptions,
  EntityLinkOptions,
  TypeAssignmentOptions,
  VoiceReferenceOptions,
  EventPrepOptions,
  AmbiguousEntityMatchOptions,
  KnowledgeGapOptions,
} from './generated/clarification-options'

export type {
  ContextAssignmentOptions,
  NewContextOptions,
  AmbiguousActionOptions,
  AmbiguousDeadlineOptions,
  EntityLinkOptions,
  TypeAssignmentOptions,
  VoiceReferenceOptions,
  EventPrepOptions,
  AmbiguousEntityMatchOptions,
  KnowledgeGapOptions,
}

export type ClarificationAnswerOptions =
  | ContextAssignmentOptions
  | NewContextOptions
  | AmbiguousActionOptions
  | AmbiguousDeadlineOptions
  | EntityLinkOptions
  | TypeAssignmentOptions
  | VoiceReferenceOptions
  | EventPrepOptions
  | AmbiguousEntityMatchOptions
  | KnowledgeGapOptions
  | null

export interface ClarificationItem {
  id: string
  kind: ClarificationKind
  status: ClarificationStatus
  subjectType: string
  subjectId: string
  subjectDescription: string
  question: string
  claudeGuess?: Record<string, unknown>
  reasoning?: string
  answerOptions: ClarificationAnswerOptions
  answer?: Record<string, unknown>
  priorityScore: number
  snoozedUntil?: string
  createdAt: string
  resolvedAt?: string
}

export interface ClarificationCountResponse {
  count: number
}
