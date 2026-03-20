import { onMounted, computed } from 'vue'
import { useContextStore } from '@/stores/contextStore'
import { useTagStore } from '@/stores/tagStore'
import { useTaskStore } from '@/stores/taskStore'
import { storeToRefs } from 'pinia'
import type { UpdateContext, NewEvent } from '@/types'

export function useContextDetail(contextId: string) {
  const contextStore = useContextStore()
  const tagStore = useTagStore()
  const taskStore = useTaskStore()
  const { currentItem: currentContext, events, eventsTotal, loading } = storeToRefs(contextStore)

  const tags = computed(() => tagStore.contextTags[contextId] ?? [])
  const linkedTasks = computed(() => taskStore.items.filter((t) => t.contextId === contextId))

  async function load() {
    await Promise.all([
      contextStore.fetchById(contextId),
      contextStore.fetchEvents(contextId),
      tagStore.fetchTagsForContext(contextId),
      taskStore.fetchList(true),
    ])
  }

  async function update(data: UpdateContext) {
    return contextStore.update(contextId, data)
  }

  async function remove() {
    return contextStore.remove(contextId)
  }

  async function addEvent(event: NewEvent) {
    return contextStore.addEvent(contextId, event)
  }

  async function addTag(tagId: string) {
    return tagStore.addTagToContext(contextId, tagId)
  }

  async function removeTag(tagId: string) {
    return tagStore.removeTagFromContext(contextId, tagId)
  }

  onMounted(load)

  return {
    context: currentContext,
    events,
    eventsTotal,
    tags,
    linkedTasks,
    loading,
    update,
    remove,
    addEvent,
    addTag,
    removeTag,
    reload: load,
  }
}
