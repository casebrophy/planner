import type { ContextStatus, ContextKind } from './enums'

export interface Context {
  id: string
  title: string
  description: string
  kind: ContextKind
  status: ContextStatus
  summary: string
  lastEvent?: string
  createdAt: string
  updatedAt: string
  parentContextId?: string
}

export interface NewContext {
  title: string
  description: string
  kind?: ContextKind
  parentContextId?: string
}

export interface UpdateContext {
  title?: string
  description?: string
  kind?: ContextKind
  status?: ContextStatus
  summary?: string
  parentContextId?: string
}

export interface ContextFilter {
  status?: ContextStatus
  kind?: ContextKind
  title?: string
  parentContextId?: string
}
