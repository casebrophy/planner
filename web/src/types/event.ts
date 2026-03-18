export interface ContextEvent {
  id: string
  contextId: string
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
  createdAt: string
}

export interface NewEvent {
  kind: string
  content: string
  metadata?: Record<string, unknown>
  sourceId?: string
}
