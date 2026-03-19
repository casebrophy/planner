import type { ContextStatus } from './enums'

export interface Context {
  id: string
  title: string
  description: string
  status: ContextStatus
  summary: string
  lastEvent?: string
  createdAt: string
  updatedAt: string
}

export interface NewContext {
  title: string
  description: string
}

export interface UpdateContext {
  title?: string
  description?: string
  status?: ContextStatus
  summary?: string
}

export interface ContextFilter {
  status?: ContextStatus
  title?: string
}
