import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useTaskNotes } from '@/composables/useTaskNotes'
import { makeNote, makeQueryResult } from '../helpers/testFactories'
import { noteService } from '@/services/noteService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/noteService', () => ({
  noteService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

function withSetup<T>(composable: () => T) {
  let result!: T
  const wrapper = mount(
    defineComponent({
      setup() {
        result = composable()
        return {}
      },
      template: '<div />',
    }),
    { global: { plugins: [createPinia()] } },
  )
  return { result, wrapper }
}

describe('useTaskNotes', () => {
  let wrapper: ReturnType<typeof mount>

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('loads notes filtered by taskId on mount', async () => {
    const taskId = 'task-123'
    const note1 = makeNote({ taskId })
    const note2 = makeNote({ taskId })
    vi.mocked(noteService.list).mockResolvedValue(
      makeQueryResult([note1, note2]),
    )

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(noteService.list).toHaveBeenCalledWith(expect.objectContaining({
      filter: { taskId },
    }))
  })

  it('returns empty notes array when no notes match taskId', async () => {
    const taskId = 'task-123'
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(setup.result.notes.value).toEqual([])
  })

  it('addNote creates a note', async () => {
    const taskId = 'task-123'
    const newNote = { content: 'Test note', source: 'manual' }
    const createdNote = makeNote({ ...newNote, taskId })
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(noteService.create).mockResolvedValue(createdNote)

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.addNote(newNote)

    expect(noteService.create).toHaveBeenCalledWith(newNote)
  })

  it('updateNote delegates to service', async () => {
    const taskId = 'task-123'
    const note = makeNote({ taskId })
    const updatedNote = makeNote({ id: note.id, taskId, content: 'Updated' })
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))
    vi.mocked(noteService.update).mockResolvedValue(updatedNote)

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.updateNote(note.id, { content: 'Updated' })

    expect(noteService.update).toHaveBeenCalledWith(note.id, { content: 'Updated' })
  })

  it('deleteNote delegates to service', async () => {
    const taskId = 'task-123'
    const note = makeNote({ taskId })
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))
    vi.mocked(noteService.delete).mockResolvedValue(undefined)

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.deleteNote(note.id)

    expect(noteService.delete).toHaveBeenCalledWith(note.id)
  })

  it('reload re-fetches notes for taskId', async () => {
    const taskId = 'task-123'
    const note = makeNote({ taskId })
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    vi.clearAllMocks()
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))

    await setup.result.reload()

    expect(noteService.list).toHaveBeenCalledWith(expect.objectContaining({
      filter: { taskId },
    }))
  })
})
