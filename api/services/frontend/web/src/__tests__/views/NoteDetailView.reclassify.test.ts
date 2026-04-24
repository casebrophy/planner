import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import NoteDetailView from '@/views/NoteDetailView.vue'
import { makeNote, makeTask } from '../helpers/testFactories'

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

// Mock the context store
vi.mock('@/stores/contextStore', () => ({
  useContextStore: () => ({
    items: ref([]),
    fetchList: vi.fn().mockResolvedValue(undefined),
  }),
}))

// Mock the entity link store
vi.mock('@/stores/entityLinkStore', () => ({
  useEntityLinkStore: () => ({
    fetchLinks: vi.fn().mockResolvedValue(undefined),
    getLinks: vi.fn(() => []),
    createLink: vi.fn().mockResolvedValue(undefined),
    deleteLink: vi.fn().mockResolvedValue(undefined),
  }),
}))

// Mock the tag store
vi.mock('@/stores/tagStore', () => ({
  useTagStore: () => ({
    create: vi.fn().mockResolvedValue({ id: 'tag-1', name: 'test-tag' }),
  }),
}))

// Mock the note store
vi.mock('@/stores/noteStore', () => ({
  useNoteStore: () => ({
    remove: vi.fn().mockResolvedValue(undefined),
  }),
}))

// Mock the correction service
vi.mock('@/services/correctionService', () => ({
  correctionService: {
    correct: vi.fn().mockResolvedValue(undefined),
  },
}))

// Mock the note service
const { mockConvertNoteToTask } = vi.hoisted(() => ({
  mockConvertNoteToTask: vi.fn(),
}))

vi.mock('@/services/noteService', () => ({
  noteService: {
    convertNoteToTask: mockConvertNoteToTask,
  },
}))

// Mock useNoteDetail
vi.mock('@/composables/useNoteDetail', () => ({
  useNoteDetail: vi.fn((noteId: string) => ({
    note: ref(makeNote({ id: noteId })),
    tags: ref([]),
    loading: ref(false),
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    addTag: vi.fn().mockResolvedValue(undefined),
    removeTag: vi.fn().mockResolvedValue(undefined),
  })),
}))

// Mock useRelatedByContext
vi.mock('@/composables/useRelatedByContext', () => ({
  useRelatedByContext: vi.fn(() => ({
    tasks: ref([]),
    notes: ref([]),
    loading: ref(false),
  })),
}))

describe('NoteDetailView - Convert to Task', () => {
  let router: ReturnType<typeof createRouter>
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    pinia = createPinia()
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/notes/:id',
          name: 'note-detail',
          component: NoteDetailView,
        },
        {
          path: '/tasks/:id',
          name: 'task-detail',
          component: { template: '<div>Task Detail</div>' },
        },
      ],
    })
    vi.clearAllMocks()
  })

  it('shows convert button in the UI', async () => {
    await router.push({ path: '/notes/test-note-1' })

    const wrapper = mount(NoteDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          LoadingSpinner: true,
          ThreadPanel: true,
          ConfirmDialog: { template: '<div><slot /></div>' },
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    })

    await wrapper.vm.$nextTick()
    const buttons = wrapper.findAll('button')
    const convertButton = buttons.find(b => b.text().includes('Convert to Task'));
    expect(convertButton).toBeDefined();
  });

  it('opens confirmation dialog when convert button is clicked', async () => {
    await router.push({ path: '/notes/test-note-1' })

    const wrapper = mount(NoteDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          LoadingSpinner: true,
          ThreadPanel: true,
          ConfirmDialog: { template: '<div><slot /></div>' },
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    })

    await wrapper.vm.$nextTick()
    const buttons = wrapper.findAll('button')
    const convertButton = buttons.find(b => b.text().includes('Convert to Task'))
    expect(convertButton).toBeDefined()

    await convertButton!.trigger('click');
    await wrapper.vm.$nextTick();

    expect((wrapper.vm as any).showConvertConfirm).toBe(true);
  });

  it('calls noteService.convertNoteToTask on confirm', async () => {
    const newTask = makeTask({ id: 'new-task-1' })
    mockConvertNoteToTask.mockResolvedValue(newTask)

    await router.push({ path: '/notes/test-note-1' })

    const wrapper = mount(NoteDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          LoadingSpinner: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    })

    await wrapper.vm.$nextTick();
    (wrapper.vm as any).showConvertConfirm = true;
    await wrapper.vm.$nextTick();

    await (wrapper.vm as any).handleConvertToTask();
    await flushPromises();

    expect(mockConvertNoteToTask).toHaveBeenCalledWith('test-note-1');
  });

  it('navigates to task detail on successful conversion', async () => {
    const newTask = makeTask({ id: 'new-task-1' })
    mockConvertNoteToTask.mockResolvedValue(newTask)

    const routerPushSpy = vi.spyOn(router, 'push')

    await router.push({ path: '/notes/test-note-1' })

    const wrapper = mount(NoteDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          LoadingSpinner: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    })

    await wrapper.vm.$nextTick();
    await (wrapper.vm as any).handleConvertToTask();
    await flushPromises();

    expect(routerPushSpy).toHaveBeenCalledWith({
      name: 'task-detail',
      params: { id: 'new-task-1' },
    });
  });

  it('preserves context in navigation query', async () => {
    const newTask = makeTask({ id: 'new-task-1' })
    mockConvertNoteToTask.mockResolvedValue(newTask)

    const routerPushSpy = vi.spyOn(router, 'push')

    await router.push({ path: '/notes/test-note-1', query: { context: 'ctx-456' } })

    const wrapper = mount(NoteDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          NoteForm: true,
          TagList: true,
          TagPicker: true,
          LoadingSpinner: true,
          ThreadPanel: true,
          ConfirmDialog: true,
          ActivityLogButton: true,
          StreakDisplay: true,
          ActivityHistory: true,
        },
      },
    })

    await wrapper.vm.$nextTick();
    await (wrapper.vm as any).handleConvertToTask();
    await flushPromises();

    expect(routerPushSpy).toHaveBeenCalledWith({
      name: 'task-detail',
      params: { id: 'new-task-1' },
      query: { context: 'ctx-456' },
    });
  });
});
