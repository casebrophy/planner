import { describe, it, expect, beforeEach, vi } from 'vitest'
import { ref, nextTick } from 'vue'
import { useRelatedByContext } from '@/composables/useRelatedByContext'

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
  },
}))

vi.mock('@/services/noteService', () => ({
  noteService: {
    list: vi.fn().mockResolvedValue({ items: [], total: 0 }),
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
    const mockTasks = [
      { id: 'task-2', title: 'Other task', contextId: 'ctx-1' },
      { id: 'task-1', title: 'Self', contextId: 'ctx-1' },
    ]
    const mockNotes = [
      { id: 'note-1', content: 'A note', contextId: 'ctx-1' },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 2 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 1 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'task', 'task-1')

    // Wait for both API calls to resolve
    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    // Self is excluded from tasks
    expect(tasks.value).toEqual([{ id: 'task-2', title: 'Other task', contextId: 'ctx-1' }])
    expect(notes.value).toEqual(mockNotes)
  })

  it('excludes self from notes when entityType is note', async () => {
    const mockTasks = [{ id: 'task-1', title: 'A task', contextId: 'ctx-1' }]
    const mockNotes = [
      { id: 'note-1', content: 'Self note', contextId: 'ctx-1' },
      { id: 'note-2', content: 'Other note', contextId: 'ctx-1' },
    ]
    vi.mocked(taskService.list).mockResolvedValueOnce({ items: mockTasks, total: 1 })
    vi.mocked(noteService.list).mockResolvedValueOnce({ items: mockNotes, total: 2 })

    const { tasks, notes, loading } = useRelatedByContext(ref('ctx-1'), 'note', 'note-1')

    await vi.waitFor(() => {
      expect(loading.value).toBe(false)
    })

    expect(tasks.value).toEqual(mockTasks)
    // Self excluded from notes
    expect(notes.value).toEqual([{ id: 'note-2', content: 'Other note', contextId: 'ctx-1' }])
  })

  it('refetches when contextId changes', async () => {
    vi.mocked(taskService.list).mockResolvedValue({ items: [], total: 0 })
    vi.mocked(noteService.list).mockResolvedValue({ items: [], total: 0 })

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
