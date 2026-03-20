import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createPinia } from 'pinia'
import { mount } from '@vue/test-utils'
import { defineComponent, nextTick } from 'vue'
import { useTaskDetail } from '@/composables/useTaskDetail'
import { useTagStore } from '@/stores/tagStore'
import { makeTask, makeTag } from '../helpers/testFactories'
import { taskService } from '@/services/taskService'
import { tagService } from '@/services/tagService'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
}))

vi.mock('@/services/taskService', () => ({
  taskService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('@/services/tagService', () => ({
  tagService: {
    list: vi.fn(),
    getById: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    getByTask: vi.fn(),
    addToTask: vi.fn(),
    removeFromTask: vi.fn(),
    getByContext: vi.fn(),
    addToContext: vi.fn(),
    removeFromContext: vi.fn(),
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

describe('useTaskDetail', () => {
  let wrapper: ReturnType<typeof mount>

  beforeEach(() => {
    vi.clearAllMocks()
  })

  afterEach(() => {
    wrapper?.unmount()
  })

  it('loads task and tags on mount', async () => {
    const task = makeTask()
    const tag = makeTag()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([tag])

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    // wait for both async calls to settle
    await nextTick()

    expect(taskService.getById).toHaveBeenCalledWith(task.id)
    expect(tagService.getByTask).toHaveBeenCalledWith(task.id)
    expect(setup.result.task.value).toEqual(task)
    expect(setup.result.tags.value).toEqual([tag])
  })

  it('update delegates to taskStore.update with taskId', async () => {
    const task = makeTask()
    const updatedTask = makeTask({ id: task.id, title: 'new title' })
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([])
    vi.mocked(taskService.update).mockResolvedValue(updatedTask)

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.update({ title: 'new title' })

    expect(taskService.update).toHaveBeenCalledWith(task.id, { title: 'new title' })
  })

  it('remove delegates to taskStore.remove with taskId', async () => {
    const task = makeTask()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([])
    vi.mocked(taskService.delete).mockResolvedValue(undefined)

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.remove()

    expect(taskService.delete).toHaveBeenCalledWith(task.id)
  })

  it('addTag delegates to tagStore.addTagToTask with taskId and tagId', async () => {
    const task = makeTask()
    const tag = makeTag()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([])
    vi.mocked(tagService.addToTask).mockResolvedValue(undefined)

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    // seed the tag into the tag store so addTagToTask can find it by id
    const tagStore = useTagStore()
    tagStore.items = [tag]

    await setup.result.addTag(tag.id)

    expect(tagService.addToTask).toHaveBeenCalledWith(task.id, tag.id)
  })

  it('removeTag delegates to tagStore.removeTagFromTask with taskId and tagId', async () => {
    const task = makeTask()
    const tag = makeTag()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([tag])
    vi.mocked(tagService.removeFromTask).mockResolvedValue(undefined)

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    await setup.result.removeTag(tag.id)

    expect(tagService.removeFromTask).toHaveBeenCalledWith(task.id, tag.id)
  })

  it('tags computed returns empty array when no taskTags entry exists', async () => {
    const task = makeTask()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([])

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    // access tags before any async resolution to confirm the default
    const tagStore = useTagStore()
    // clear any entry that may have been set during mount
    delete tagStore.taskTags[task.id]

    expect(setup.result.tags.value).toEqual([])
  })

  it('reload re-fetches task and tags', async () => {
    const task = makeTask()
    const tag = makeTag()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([tag])

    const setup = withSetup(() => useTaskDetail(task.id))
    wrapper = setup.wrapper

    await nextTick()
    await nextTick()

    // clear call counts then reload
    vi.clearAllMocks()
    vi.mocked(taskService.getById).mockResolvedValue(task)
    vi.mocked(tagService.getByTask).mockResolvedValue([tag])

    await setup.result.reload()

    expect(taskService.getById).toHaveBeenCalledTimes(1)
    expect(taskService.getById).toHaveBeenCalledWith(task.id)
    expect(tagService.getByTask).toHaveBeenCalledTimes(1)
    expect(tagService.getByTask).toHaveBeenCalledWith(task.id)
  })
})
