import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick, ref } from 'vue'
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
    const note1 = makeNote()
    const note2 = makeNote()
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
    const createdNote = makeNote({ ...newNote })
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
    const note = makeNote()
    const updatedNote = makeNote({ id: note.id, content: 'Updated' })
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
    const note = makeNote()
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
    const note = makeNote()
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

  it('does not call fetchList when taskId is empty string', async () => {
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useTaskNotes(''))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(noteService.list).not.toHaveBeenCalled()
  })

  it('uses the ref value when taskId is a Ref<string>', async () => {
    const taskId = ref('task-ref-456')
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([]))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    expect(noteService.list).toHaveBeenCalledWith(expect.objectContaining({
      filter: { taskId: 'task-ref-456' },
    }))
  })

  it('propagates error when addNote (create) rejects', async () => {
    const taskId = 'task-123'
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([]))
    vi.mocked(noteService.create).mockRejectedValue(new Error('create failed'))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await expect(setup.result.addNote({ content: 'test', source: 'manual' })).rejects.toThrow('create failed')
  })

  it('propagates error when updateNote (update) rejects', async () => {
    const taskId = 'task-123'
    const note = makeNote()
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))
    vi.mocked(noteService.update).mockRejectedValue(new Error('update failed'))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await expect(setup.result.updateNote(note.id, { content: 'changed' })).rejects.toThrow('update failed')
  })

  it('propagates error when deleteNote (remove) rejects', async () => {
    const taskId = 'task-123'
    const note = makeNote()
    vi.mocked(noteService.list).mockResolvedValue(makeQueryResult([note]))
    vi.mocked(noteService.delete).mockRejectedValue(new Error('delete failed'))

    const setup = withSetup(() => useTaskNotes(taskId))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await expect(setup.result.deleteNote(note.id)).rejects.toThrow('delete failed')
  })
})
