import type { Task, Context, Tag } from '@/types'
import { TaskStatus, TaskPriority, TaskEnergy, ContextStatus } from '@/types'

let counter = 0
function uid(): string {
  return `test-${++counter}-${Date.now()}`
}

export function resetFactoryCounter() {
  counter = 0
}

export function makeTask(overrides: Partial<Task> = {}): Task {
  const id = uid()
  return {
    id,
    title: `Task ${id}`,
    description: '',
    status: TaskStatus.Todo,
    priority: TaskPriority.Medium,
    energy: TaskEnergy.Medium,
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeContext(overrides: Partial<Context> = {}): Context {
  const id = uid()
  return {
    id,
    title: `Context ${id}`,
    description: '',
    status: ContextStatus.Active,
    summary: '',
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeTag(overrides: Partial<Tag> = {}): Tag {
  const id = uid()
  return {
    id,
    name: `tag-${id}`,
    ...overrides,
  }
}

export function makeQueryResult<T>(items: T[], total?: number) {
  return {
    items,
    total: total ?? items.length,
    page: 1,
    rowsPerPage: 20,
  }
}
