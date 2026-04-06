import type { Task, Context, Tag, ClarificationItem, ContextEvent, TimeBlock } from '@/types'
import { TaskStatus, TaskPriority, TaskEnergy, ContextStatus, ContextKind, ClarificationKind, ClarificationStatus } from '@/types'

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
    status: TaskStatus.Open,
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
    kind: ContextKind.Project,
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

export function makeClarificationItem(overrides: Partial<ClarificationItem> = {}): ClarificationItem {
  const id = uid()
  return {
    id,
    kind: ClarificationKind.StaleTask,
    status: ClarificationStatus.Pending,
    subjectType: 'task',
    subjectId: uid(),
    question: `Clarification ${id}?`,
    answerOptions: null,
    priorityScore: 50,
    createdAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeContextEvent(overrides: Partial<ContextEvent> = {}): ContextEvent {
  const id = uid()
  return {
    id,
    contextId: uid(),
    kind: 'note',
    content: `Event ${id}`,
    createdAt: new Date().toISOString(),
    ...overrides,
  }
}

export function makeTimeBlock(overrides: Partial<TimeBlock> = {}): TimeBlock {
  const id = uid()
  const now = new Date().toISOString()
  const start = new Date(Date.now() + 3600000).toISOString() // 1 hour from now
  const end = new Date(Date.now() + 7200000).toISOString() // 2 hours from now
  return {
    id,
    taskId: uid(),
    startsAt: start,
    endsAt: end,
    confirmed: false,
    createdAt: now,
    updatedAt: now,
    ...overrides,
  }
}
