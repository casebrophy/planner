import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { useRelatedByContext } from '@/composables/useRelatedByContext'
import type { Task } from '@/types/task'
import type { Note } from '@/types/note'

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 }),
  },
}))

vi.mock('@/services/noteService', () => ({
  noteService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 }),
  },
}))

import { taskService } from '@/services/taskService'
import { noteService } from '@/services/noteService'

describe('useRelatedByContext', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns empty arrays when contextId is undefined', async () => {
    const { tasks, notes } = useRelatedByContext(ref(undefined), 'task', 'task-1')
    await nextTick()
    expect(tasks.value).toEqual([])
    expect(notes.value).toEqual([])
    expect(taskService.list).not.toHaveBeenCalled()
    expect(noteService.list).not.toHaveBeenCalled()
  })

  it('fetches tasks and notes when contextId is set', async () => {
    const mockTasks: Task[] = [
      {
        id: 'task-2',
        title: 'Other task',
        description: '',
        status: 'open' as Task['status'],
        priority: 'medium' as Task['priority'],
        energy: 'medium' as Task['energy'],
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
      {
        id: 'task-1',
        title: 'Self',
        description: '',
        status: 'open' as Task['status'],
        priority: 'medium' as Task['priority'],
        energy: 'medium' as Task['energy'],
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
    ]
    const mockNotes: Note[] = [
      {
        id: 'note-1',
        content: 'A note',
        source: 'manual',
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 2, page: 1, rowsPerPage: 100 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 1, page: 1, rowsPerPage: 100 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'task', 'task-1')

    // Wait for both API calls to resolve
    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    // Self is excluded from tasks
    expect(tasks.value).toEqual([mockTasks[0]])
    expect(notes.value).toEqual(mockNotes)
  })

  it('excludes self from notes when entityType is note', async () => {
    const mockTasks: Task[] = [
      {
        id: 'task-1',
        title: 'A task',
        description: '',
        status: 'open' as Task['status'],
        priority: 'medium' as Task['priority'],
        energy: 'medium' as Task['energy'],
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
    ]
    const mockNotes: Note[] = [
      {
        id: 'note-1',
        content: 'Self note',
        source: 'manual',
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
      {
        id: 'note-2',
        content: 'Other note',
        source: 'manual',
        contextId: 'ctx-1',
        createdAt: '2024-01-01T00:00:00Z',
        updatedAt: '2024-01-01T00:00:00Z',
      },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 1, page: 1, rowsPerPage: 100 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 2, page: 1, rowsPerPage: 100 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'note', 'note-1')

    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    expect(tasks.value).toEqual(mockTasks)
    // Self excluded from notes
    expect(notes.value).toEqual([mockNotes[1]])
  })

  it('refetches when contextId changes', async () => {
    vi.mocked(taskService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 })
    vi.mocked(noteService.list).mockResolvedValue({ items: [], total: 0, page: 1, rowsPerPage: 100 })

    const contextId = ref<string | undefined>('ctx-1')
    const { loading } = useRelatedByContext(contextId, 'task', 'task-1')

    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })
    expect(taskService.list).toHaveBeenCalledTimes(1)

    contextId.value = 'ctx-2'
    await nextTick()
    await vi.waitFor(() => {
      expect(taskService.list).toHaveBeenCalledTimes(2)
    })
  })
})
