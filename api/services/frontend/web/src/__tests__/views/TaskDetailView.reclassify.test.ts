import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { ref } from 'vue'
import { createPinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import TaskDetailView from '@/views/TaskDetailView.vue'
import { makeTask, makeNote } from '../helpers/testFactories'

// Mock the toast store
vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
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

// Mock the task store
vi.mock('@/stores/taskStore', () => ({
  useTaskStore: () => ({
    remove: vi.fn().mockResolvedValue(undefined),
  }),
}))

// Mock the observation service
vi.mock('@/services/observationService', () => ({
  observationService: {
    queryByKind: vi.fn().mockResolvedValue([]),
    record: vi.fn().mockResolvedValue(undefined),
    queryBySubject: vi.fn().mockResolvedValue([]),
  },
}))

// Mock the correction service
vi.mock('@/services/correctionService', () => ({
  correctionService: {
    correct: vi.fn().mockResolvedValue(undefined),
  },
}))

// Mock the task service
const { mockConvertTaskToNote } = vi.hoisted(() => ({
  mockConvertTaskToNote: vi.fn(),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    convertTaskToNote: mockConvertTaskToNote,
  },
}))

// Mock useTaskDetail
vi.mock('@/composables/useTaskDetail', () => ({
  useTaskDetail: vi.fn((taskId: string) => ({
    task: ref(makeTask({ id: taskId })),
    tags: ref([]),
    loading: ref(false),
    update: vi.fn().mockResolvedValue(undefined),
    remove: vi.fn().mockResolvedValue(undefined),
    addTag: vi.fn().mockResolvedValue(undefined),
    removeTag: vi.fn().mockResolvedValue(undefined),
  })),
}))

// Mock useTaskNotes
vi.mock('@/composables/useTaskNotes', () => ({
  useTaskNotes: vi.fn(() => ({
    notes: ref([]),
    loading: ref(false),
    addNote: vi.fn().mockResolvedValue(undefined),
    updateNote: vi.fn().mockResolvedValue(undefined),
    deleteNote: vi.fn().mockResolvedValue(undefined),
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

describe('TaskDetailView - Convert to Note', () => {
  let router: ReturnType<typeof createRouter>
  let pinia: ReturnType<typeof createPinia>

  beforeEach(() => {
    pinia = createPinia()
    router = createRouter({
      history: createMemoryHistory(),
      routes: [
        {
          path: '/tasks/:id',
          name: 'task-detail',
          component: TaskDetailView,
        },
        {
          path: '/notes/:id',
          name: 'note-detail',
          component: { template: '<div>Note Detail</div>' },
        },
      ],
    })
    vi.clearAllMocks()
  })

  it('shows convert button in the UI', async () => {
    await router.push({ path: '/tasks/test-task-1' })

    const wrapper = mount(TaskDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          TaskForm: true,
          TaskDebriefDialog: true,
          TagList: true,
          TagPicker: true,
          NoteList: true,
          NoteForm: true,
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
    const convertButton = buttons.find(b => b.text().includes('Convert to Note'));
    expect(convertButton).toBeDefined();
  });

  it('opens confirmation dialog when convert button is clicked', async () => {
    await router.push({ path: '/tasks/test-task-1' })

    const wrapper = mount(TaskDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          TaskForm: true,
          TaskDebriefDialog: true,
          TagList: true,
          TagPicker: true,
          NoteList: true,
          NoteForm: true,
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
    const convertButton = buttons.find(b => b.text().includes('Convert to Note'))
    expect(convertButton).toBeDefined()

    await convertButton!.trigger('click');
    await wrapper.vm.$nextTick();

    expect((wrapper.vm as any).showConvertConfirm).toBe(true);
  });

  it('calls taskService.convertTaskToNote on confirm', async () => {
    const newNote = makeNote({ id: 'new-note-1' })
    mockConvertTaskToNote.mockResolvedValue(newNote)

    await router.push({ path: '/tasks/test-task-1' })

    const wrapper = mount(TaskDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          TaskForm: true,
          TaskDebriefDialog: true,
          TagList: true,
          TagPicker: true,
          NoteList: true,
          NoteForm: true,
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

    await (wrapper.vm as any).handleConvertToNote();
    await flushPromises();

    expect(mockConvertTaskToNote).toHaveBeenCalledWith('test-task-1');
  });

  it('navigates to note detail on successful conversion', async () => {
    const newNote = makeNote({ id: 'new-note-1' })
    mockConvertTaskToNote.mockResolvedValue(newNote)

    const routerPushSpy = vi.spyOn(router, 'push')

    await router.push({ path: '/tasks/test-task-1' })

    const wrapper = mount(TaskDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          TaskForm: true,
          TaskDebriefDialog: true,
          TagList: true,
          TagPicker: true,
          NoteList: true,
          NoteForm: true,
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
    await (wrapper.vm as any).handleConvertToNote();
    await flushPromises();

    expect(routerPushSpy).toHaveBeenCalledWith({
      name: 'note-detail',
      params: { id: 'new-note-1' },
    });
  });

  it('preserves context in navigation query', async () => {
    const newNote = makeNote({ id: 'new-note-1' })
    mockConvertTaskToNote.mockResolvedValue(newNote)

    const routerPushSpy = vi.spyOn(router, 'push')

    await router.push({ path: '/tasks/test-task-1', query: { context: 'ctx-123' } })

    const wrapper = mount(TaskDetailView, {
      global: {
        plugins: [pinia, router],
        stubs: {
          TaskForm: true,
          TaskDebriefDialog: true,
          TagList: true,
          TagPicker: true,
          NoteList: true,
          NoteForm: true,
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
    await (wrapper.vm as any).handleConvertToNote();
    await flushPromises();

    expect(routerPushSpy).toHaveBeenCalledWith({
      name: 'note-detail',
      params: { id: 'new-note-1' },
      query: { context: 'ctx-123' },
    });
  });
});
