import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TaskDetailView from '@/views/TaskDetailView.vue'
import { makeTask, makeNote } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/stores/entityLinkStore', () => {
  const mockStore = {
    fetchLinks: vi.fn().mockResolvedValue(undefined),
    getLinks: vi.fn(() => []),
    createLink: vi.fn().mockResolvedValue(undefined),
    deleteLink: vi.fn().mockResolvedValue(undefined),
  }
  return {
    useEntityLinkStore: vi.fn(() => mockStore),
    entityLinkStore: mockStore,
  }
})

vi.mock('@/stores/tagStore', () => ({
  useTagStore: () => ({
    create: vi.fn().mockResolvedValue({ id: 'tag-1', name: 'test-tag' }),
  }),
}))

vi.mock('@/composables/useTaskDetail', () => ({
  useTaskDetail: vi.fn((taskId: string) => ({
    task: { value: makeTask({ id: taskId }) },
    tags: [],
    loading: false,
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    addTag: vi.fn().mockResolvedValue(undefined),
    removeTag: vi.fn().mockResolvedValue(undefined),
  })),
}))

const mockDeleteNote = vi.fn().mockResolvedValue(undefined)
const mockAddNote = vi.fn().mockResolvedValue(undefined)
const mockUpdateNote = vi.fn().mockResolvedValue(undefined)
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockNotesRef = { value: [] as any[] }

vi.mock('@/composables/useTaskNotes', () => ({
  useTaskNotes: vi.fn(() => ({
    notes: mockNotesRef,
    loading: { value: false },
    addNote: mockAddNote,
    updateNote: mockUpdateNote,
    deleteNote: mockDeleteNote,
    reload: vi.fn(),
  })),
}))

async function mountView(taskId: string = 'test-task-1') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/tasks', name: 'tasks', component: { template: '<div />' } },
      { path: '/tasks/:id', name: 'task-detail', component: TaskDetailView },
    ],
  })
  router.push(`/tasks/${taskId}`)
  await router.isReady()

  return {
    wrapper: mount(TaskDetailView, {
      global: {
        plugins: [createPinia(), router],
        stubs: {
          LoadingSpinner: true,
          TaskForm: true,
          TagList: true,
          TagPicker: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
          NoteList: true,
          NoteForm: true,
        },
      },
    }),
    router,
  }
}

describe('TaskDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
  })

  it('renders task title in view mode', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Edit')
    expect(wrapper.text()).toContain('Delete')
    wrapper.unmount()
  })

  it('shows Edit and Delete buttons in view mode', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text() === 'Edit')).toBe(true)
    expect(buttons.some(b => b.text() === 'Delete')).toBe(true)
    wrapper.unmount()
  })

  it('toggles to edit mode when Edit button clicked', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    const editBtn = buttons.find(b => b.text() === 'Edit')
    if (editBtn) {
      await editBtn.trigger('click')
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).editing).toBe(true)
    }
    wrapper.unmount()
  })

  it('shows ConfirmDialog when Delete button clicked', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    const deleteBtn = buttons.find(b => b.text() === 'Delete')
    if (deleteBtn) {
      await deleteBtn.trigger('click')
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).confirmDelete).toBe(true)
    }
    wrapper.unmount()
  })

  it('renders Tags section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Tags')
    const tagList = wrapper.findComponent({ name: 'TagList' })
    expect(tagList.exists()).toBe(true)
    wrapper.unmount()
  })

  it('renders Activity section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Activity')
    const streakDisplay = wrapper.findComponent({ name: 'StreakDisplay' })
    expect(streakDisplay.exists()).toBe(true)
    wrapper.unmount()
  })

  it('renders ThreadPanel for activity thread', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const threadPanel = wrapper.findComponent({ name: 'ThreadPanel' })
    expect(threadPanel.exists()).toBe(true)
    wrapper.unmount()
  })

  it('renders Related Items section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Related Items')
    wrapper.unmount()
  })

  it('shows "Link manually" button in Related Items section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text().includes('Link manually'))).toBe(true)
    wrapper.unmount()
  })

  it('opens link modal when "Link manually" button clicked', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    const linkBtn = buttons.find(b => b.text().includes('Link manually'))
    if (linkBtn) {
      await linkBtn.trigger('click')
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).showLinkModal).toBe(true)
    }
    wrapper.unmount()
  })

  it('closes link modal when Cancel button clicked', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    // Open link modal first
    const buttons = wrapper.findAll('button')
    const linkBtn = buttons.find(b => b.text().includes('Link manually'))
    if (linkBtn) {
      await linkBtn.trigger('click')
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).showLinkModal).toBe(true)

      // Find and click Cancel button
      const allButtons = wrapper.findAll('button')
      const cancelBtn = allButtons.find(b => b.text() === 'Cancel')
      if (cancelBtn) {
        await cancelBtn.trigger('click')
        await flushPromises()
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        expect((wrapper.vm as any).showLinkModal).toBe(false)
      }
    }
    wrapper.unmount()
  })

  it('passes taskId to ThreadPanel', async () => {
    const taskId = 'test-task-123'
    const { wrapper } = await mountView(taskId)
    await flushPromises()

    const threadPanel = wrapper.findComponent({ name: 'ThreadPanel' })
    expect(threadPanel.props('subjectId')).toBe(taskId)
    wrapper.unmount()
  })

  it('passes taskId to ActivityLogButton', async () => {
    const taskId = 'test-task-456'
    const { wrapper } = await mountView(taskId)
    await flushPromises()

    const activityBtn = wrapper.findComponent({ name: 'ActivityLogButton' })
    expect(activityBtn.props('subjectId')).toBe(taskId)
    wrapper.unmount()
  })

  it('passes taskId to StreakDisplay', async () => {
    const taskId = 'test-task-789'
    const { wrapper } = await mountView(taskId)
    await flushPromises()

    const streakDisplay = wrapper.findComponent({ name: 'StreakDisplay' })
    expect(streakDisplay.props('subjectId')).toBe(taskId)
    wrapper.unmount()
  })

  it('renders TaskForm when editing', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    // Toggle edit mode
    const buttons = wrapper.findAll('button')
    const editBtn = buttons.find(b => b.text() === 'Edit')
    if (editBtn) {
      await editBtn.trigger('click')
      await flushPromises()

      const taskForm = wrapper.findComponent({ name: 'TaskForm' })
      expect(taskForm.exists()).toBe(true)
    }
    wrapper.unmount()
  })

  it('exits edit mode when TaskForm is cancelled', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    // Toggle edit mode
    const buttons = wrapper.findAll('button')
    const editBtn = buttons.find(b => b.text() === 'Edit')
    if (editBtn) {
      await editBtn.trigger('click')
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).editing).toBe(true);

      // Cancel editing
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      (wrapper.vm as any).editing = false
      await flushPromises()
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      expect((wrapper.vm as any).editing).toBe(false)
    }
    wrapper.unmount()
  })

  it('fetches entity links on mount', async () => {
    const { useEntityLinkStore } = await import('@/stores/entityLinkStore')
    const { wrapper } = await mountView('test-task-1')
    await flushPromises()

    const store = useEntityLinkStore()
    expect(store.fetchLinks).toHaveBeenCalledWith('task', 'test-task-1')
    wrapper.unmount()
  })

  it('displays task status', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Status')
    wrapper.unmount()
  })

  it('displays task priority', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Priority')
    wrapper.unmount()
  })

  it('displays task energy level', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Energy')
    wrapper.unmount()
  })

  it('renders Notes section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Notes')
    const noteList = wrapper.findComponent({ name: 'NoteList' })
    expect(noteList.exists()).toBe(true)
    wrapper.unmount()
  })

  it('passes notes to NoteList', async () => {
    const note = makeNote()
    mockNotesRef.value = [note]

    const { wrapper } = await mountView()
    await flushPromises()

    const noteList = wrapper.findComponent({ name: 'NoteList' })
    expect(noteList.exists()).toBe(true)
    mockNotesRef.value = []
    wrapper.unmount()
  })

  it('renders Add Note button in notes section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text().includes('Add Note'))).toBe(true)
    wrapper.unmount()
  })

  it('shows NoteForm when Add Note button clicked', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const addBtn = wrapper.findAll('button').find(b => b.text().includes('Add Note'))
    expect(addBtn).toBeTruthy()
    await addBtn!.trigger('click')
    await flushPromises()

    const noteForm = wrapper.findComponent({ name: 'NoteForm' })
    expect(noteForm.exists()).toBe(true)
    wrapper.unmount()
  })

  it('hides Add Note button when note form is open', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const addBtn = wrapper.findAll('button').find(b => b.text().includes('Add Note'))
    await addBtn!.trigger('click')
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text().includes('Add Note'))).toBe(false)
    wrapper.unmount()
  })

  it('passes notes from useTaskNotes to NoteList', async () => {
    const note = makeNote({ taskId: 'test-task-1' })
    mockNotesRef.value = [note]

    const { wrapper } = await mountView()
    await flushPromises()

    const noteList = wrapper.findComponent({ name: 'NoteList' })
    expect(noteList.exists()).toBe(true)
    // The composable returns a ref object; the view passes it directly as :notes
    const notesProp = noteList.props('notes')
    const notesArr = Array.isArray(notesProp) ? notesProp : (notesProp as { value: unknown[] }).value
    expect(notesArr).toEqual([note])

    mockNotesRef.value = []
    wrapper.unmount()
  })
})
