import { defineStore } from 'pinia'
import { ref } from 'vue'
import { tagService } from '@/services/tagService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { Tag, NewTag } from '@/types'

export const useTagStore = defineStore('tag', () => {
  const crud = createCRUDStore<Tag, NewTag, Partial<Tag>, Record<string, never>>({
    name: 'tag',
    service: tagService,
    defaultOrderBy: 'name',
    defaultRowsPerPage: 100,
  })

  const taskTags = ref<Record<string, Tag[]>>({})
  const contextTags = ref<Record<string, Tag[]>>({})

  const toast = useToastStore()

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
    const tag = crud.items.value.find((t) => t.id === tagId)
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
    const tag = crud.items.value.find((t) => t.id === tagId)
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
    ...crud,
    taskTags,
    contextTags,
    fetchTagsForTask,
    addTagToTask,
    removeTagFromTask,
    fetchTagsForContext,
    addTagToContext,
    removeTagFromContext,
  }
})
