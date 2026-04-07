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
      expect(wrapper.vm.editing).toBe(true)
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
      expect(wrapper.vm.confirmDelete).toBe(true)
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
      expect(wrapper.vm.showLinkModal).toBe(true)
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
      expect(wrapper.vm.showLinkModal).toBe(true)

      // Find and click Cancel button
      const allButtons = wrapper.findAll('button')
      const cancelBtn = allButtons.find(b => b.text() === 'Cancel')
      if (cancelBtn) {
        await cancelBtn.trigger('click')
        await flushPromises()
        expect(wrapper.vm.showLinkModal).toBe(false)
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
      expect(wrapper.vm.editing).toBe(true)

      // Cancel editing
      wrapper.vm.editing = false
      await flushPromises()
      expect(wrapper.vm.editing).toBe(false)
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
})
