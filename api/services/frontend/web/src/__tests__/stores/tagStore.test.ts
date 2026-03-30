import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useTagStore } from '@/stores/tagStore'
import { makeTag } from '../helpers/testFactories'

vi.mock('@/stores/toastStore', () => ({
  useToastStore: () => ({ success: vi.fn(), error: vi.fn() }),
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

import { tagService } from '@/services/tagService'

describe('tagStore — association extensions', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  // ---- task tags ----

  it('fetchTagsForTask stores returned tags in taskTags keyed by taskId', async () => {
    const tag1 = makeTag()
    const tag2 = makeTag()
    vi.mocked(tagService.getByTask).mockResolvedValue([tag1, tag2])

    const store = useTagStore()
    const result = await store.fetchTagsForTask('task-abc')

    expect(tagService.getByTask).toHaveBeenCalledWith('task-abc')
    expect(store.taskTags['task-abc']).toEqual([tag1, tag2])
    expect(result).toEqual([tag1, tag2])
  })

  it('fetchTagsForTask returns empty array and shows toast on error', async () => {
    vi.mocked(tagService.getByTask).mockRejectedValue(new Error('network error'))

    const store = useTagStore()
    const result = await store.fetchTagsForTask('task-fail')

    expect(result).toEqual([])
    expect(store.taskTags['task-fail']).toBeUndefined()
  })

  it('addTagToTask calls service and appends tag to taskTags when found in items', async () => {
    const tag = makeTag()
    vi.mocked(tagService.addToTask).mockResolvedValue(undefined)

    const store = useTagStore()
    // Seed items so the store can look up the tag by id
    store.items = [tag]
    store.taskTags['task-xyz'] = []

    await store.addTagToTask('task-xyz', tag.id)

    expect(tagService.addToTask).toHaveBeenCalledWith('task-xyz', tag.id)
    expect(store.taskTags['task-xyz']).toContainEqual(tag)
  })

  it('addTagToTask initialises taskTags entry when none exists yet', async () => {
    const tag = makeTag()
    vi.mocked(tagService.addToTask).mockResolvedValue(undefined)

    const store = useTagStore()
    store.items = [tag]

    await store.addTagToTask('task-new', tag.id)

    expect(store.taskTags['task-new']).toEqual([tag])
  })

  it('removeTagFromTask calls service and filters tag out of taskTags', async () => {
    const tagA = makeTag()
    const tagB = makeTag()
    vi.mocked(tagService.removeFromTask).mockResolvedValue(undefined)

    const store = useTagStore()
    store.taskTags['task-xyz'] = [tagA, tagB]

    await store.removeTagFromTask('task-xyz', tagA.id)

    expect(tagService.removeFromTask).toHaveBeenCalledWith('task-xyz', tagA.id)
    expect(store.taskTags['task-xyz']).toEqual([tagB])
    expect(store.taskTags['task-xyz']).not.toContainEqual(tagA)
  })

  // ---- context tags ----

  it('fetchTagsForContext stores returned tags in contextTags keyed by contextId', async () => {
    const tag1 = makeTag()
    const tag2 = makeTag()
    vi.mocked(tagService.getByContext).mockResolvedValue([tag1, tag2])

    const store = useTagStore()
    const result = await store.fetchTagsForContext('ctx-abc')

    expect(tagService.getByContext).toHaveBeenCalledWith('ctx-abc')
    expect(store.contextTags['ctx-abc']).toEqual([tag1, tag2])
    expect(result).toEqual([tag1, tag2])
  })

  it('addTagToContext calls service and appends tag to contextTags when found in items', async () => {
    const tag = makeTag()
    vi.mocked(tagService.addToContext).mockResolvedValue(undefined)

    const store = useTagStore()
    store.items = [tag]

    await store.addTagToContext('ctx-xyz', tag.id)

    expect(tagService.addToContext).toHaveBeenCalledWith('ctx-xyz', tag.id)
    expect(store.contextTags['ctx-xyz']).toEqual([tag])
  })

  it('removeTagFromContext calls service and filters tag out of contextTags', async () => {
    const tagA = makeTag()
    const tagB = makeTag()
    vi.mocked(tagService.removeFromContext).mockResolvedValue(undefined)

    const store = useTagStore()
    store.contextTags['ctx-xyz'] = [tagA, tagB]

    await store.removeTagFromContext('ctx-xyz', tagA.id)

    expect(tagService.removeFromContext).toHaveBeenCalledWith('ctx-xyz', tagA.id)
    expect(store.contextTags['ctx-xyz']).toEqual([tagB])
    expect(store.contextTags['ctx-xyz']).not.toContainEqual(tagA)
  })
})
