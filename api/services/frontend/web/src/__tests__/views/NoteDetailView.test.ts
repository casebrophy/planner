import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { ref } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import NoteDetailView from '@/views/NoteDetailView.vue'
import { makeNote, makeTask } from '../helpers/testFactories'

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

vi.mock('@/stores/contextStore', () => ({
  useContextStore: () => ({
    items: [],
    fetchList: vi.fn().mockResolvedValue(undefined),
    fetchAll: vi.fn(),
  }),
}))

vi.mock('@/composables/useNoteDetail', () => ({
  useNoteDetail: vi.fn((noteId: string) => ({
    note: { value: makeNote({ id: noteId }) },
    tags: [],
    loading: false,
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    addTag: vi.fn().mockResolvedValue(undefined),
    removeTag: vi.fn().mockResolvedValue(undefined),
  })),
}))

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockRelatedTasks = ref([] as any[])
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const mockRelatedNotes = ref([] as any[])

vi.mock('@/composables/useRelatedByContext', () => ({
  useRelatedByContext: vi.fn(() => ({
    tasks: mockRelatedTasks,
    notes: mockRelatedNotes,
    loading: ref(false),
  })),
}))

async function mountView(noteId: string = 'test-note-1') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/notes', name: 'notes', component: { template: '<div />' } },
      { path: '/notes/:id', name: 'note-detail', component: NoteDetailView },
      { path: '/tasks/:id', name: 'task-detail', component: { template: '<div />' } },
    ],
  })
  router.push(`/notes/${noteId}`)
  await router.isReady()

  return {
    wrapper: mount(NoteDetailView, {
      global: {
        plugins: [createPinia(), router],
        stubs: {
          LoadingSpinner: true,
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    }),
    router,
  }
}

describe('NoteDetailView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    vi.resetAllMocks()
    mockRelatedTasks.value = []
    mockRelatedNotes.value = []
  })

  it('renders Related Items section', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).toContain('Related Items')
    wrapper.unmount()
  })

  it('fetches entity links on mount', async () => {
    const { useEntityLinkStore } = await import('@/stores/entityLinkStore')
    const { wrapper } = await mountView('test-note-1')
    await flushPromises()

    const store = useEntityLinkStore()
    expect(store.fetchLinks).toHaveBeenCalledWith('note', 'test-note-1')
    wrapper.unmount()
  })

  it('does not render "In same context" section when note has no contextId', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    expect(wrapper.text()).not.toContain('In same context')
    wrapper.unmount()
  })

  it('renders "In same context" section when related items exist', async () => {
    const { useNoteDetail } = await import('@/composables/useNoteDetail')
    vi.mocked(useNoteDetail).mockReturnValueOnce({
      note: { value: makeNote({ id: 'test-note-1', contextId: 'ctx-1' }) },
      tags: [],
      loading: false,
      update: vi.fn().mockResolvedValue(undefined),
      remove: vi.fn().mockResolvedValue(undefined),
      addTag: vi.fn().mockResolvedValue(undefined),
      removeTag: vi.fn().mockResolvedValue(undefined),
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } as any)

    mockRelatedTasks.value = [makeTask({ id: 'task-1', title: 'Context task', contextId: 'ctx-1' })]
    mockRelatedNotes.value = [makeNote({ id: 'note-2', content: 'Context note', contextId: 'ctx-1' })]

    const { wrapper } = await mountView('test-note-1')
    await flushPromises()

    expect(wrapper.text()).toContain('In same context')
    expect(wrapper.text()).toContain('Context task')
    expect(wrapper.text()).toContain('Context note')

    wrapper.unmount()
  })

  it('shows "Link manually" button', async () => {
    const { wrapper } = await mountView()
    await flushPromises()

    const buttons = wrapper.findAll('button')
    expect(buttons.some(b => b.text().includes('Link manually'))).toBe(true)
    wrapper.unmount()
  })
})
