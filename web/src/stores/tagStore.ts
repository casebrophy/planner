import { defineStore } from 'pinia'
import { ref } from 'vue'
import { tagService } from '@/services/tagService'
import { useToastStore } from './toastStore'
import type { Tag, NewTag } from '@/types'

const CACHE_TTL = 5 * 60 * 1000

export const useTagStore = defineStore('tag', () => {
  const items = ref<Tag[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastFetchedAt = ref(0)
  const taskTags = ref<Record<string, Tag[]>>({})
  const contextTags = ref<Record<string, Tag[]>>({})

  const toast = useToastStore()

  function isCacheValid(): boolean {
    return lastFetchedAt.value > 0 && Date.now() - lastFetchedAt.value < CACHE_TTL
  }

  async function fetchTags(force = false) {
    if (!force && isCacheValid()) return
    loading.value = true
    error.value = null
    try {
      const result = await tagService.list({ rows: 100 })
      items.value = result.items
      total.value = result.total
      lastFetchedAt.value = Date.now()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch tags'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function createTag(tag: NewTag) {
    try {
      const created = await tagService.create(tag)
      items.value.push(created)
      total.value++
      toast.success('Tag created')
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to create tag'
      toast.error(msg)
      throw e
    }
  }

  async function deleteTag(id: string) {
    const idx = items.value.findIndex((t) => t.id === id)
    const backup = idx !== -1 ? items.value[idx]! : null

    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
    }

    try {
      await tagService.delete(id)
      toast.success('Tag deleted')
    } catch (e) {
      if (backup && idx !== -1) {
        items.value.splice(idx, 0, backup)
        total.value++
      }
      const msg = e instanceof Error ? e.message : 'Failed to delete tag'
      toast.error(msg)
      throw e
    }
  }

  async function fetchTagsForTask(taskId: string) {
    try {
      const tags = await tagService.getByTask(taskId)
      taskTags.value[taskId] = tags
      return tags
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch task tags'
      toast.error(msg)
      return []
    }
  }

  async function addTagToTask(taskId: string, tagId: string) {
    await tagService.addToTask(taskId, tagId)
    const tag = items.value.find((t) => t.id === tagId)
    if (tag) {
      if (!taskTags.value[taskId]) taskTags.value[taskId] = []
      taskTags.value[taskId]!.push(tag)
    }
  }

  async function removeTagFromTask(taskId: string, tagId: string) {
    await tagService.removeFromTask(taskId, tagId)
    if (taskTags.value[taskId]) {
      taskTags.value[taskId] = taskTags.value[taskId]!.filter((t) => t.id !== tagId)
    }
  }

  async function fetchTagsForContext(contextId: string) {
    try {
      const tags = await tagService.getByContext(contextId)
      contextTags.value[contextId] = tags
      return tags
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch context tags'
      toast.error(msg)
      return []
    }
  }

  async function addTagToContext(contextId: string, tagId: string) {
    await tagService.addToContext(contextId, tagId)
    const tag = items.value.find((t) => t.id === tagId)
    if (tag) {
      if (!contextTags.value[contextId]) contextTags.value[contextId] = []
      contextTags.value[contextId]!.push(tag)
    }
  }

  async function removeTagFromContext(contextId: string, tagId: string) {
    await tagService.removeFromContext(contextId, tagId)
    if (contextTags.value[contextId]) {
      contextTags.value[contextId] = contextTags.value[contextId]!.filter((t) => t.id !== tagId)
    }
  }

  return {
    items,
    total,
    loading,
    error,
    lastFetchedAt,
    taskTags,
    contextTags,
    fetchTags,
    createTag,
    deleteTag,
    fetchTagsForTask,
    addTagToTask,
    removeTagFromTask,
    fetchTagsForContext,
    addTagToContext,
    removeTagFromContext,
  }
})
