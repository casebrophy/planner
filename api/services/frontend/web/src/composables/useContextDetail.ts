import { onMounted, computed, ref } from 'vue'
import { useContextStore } from '@/stores/contextStore'
import { useTagStore } from '@/stores/tagStore'
import { storeToRefs } from 'pinia'
import { fetchAllPages } from '@/services/fetchAllPages'
import { taskService } from '@/services/taskService'
import type { UpdateContext, Task } from '@/types'

export function useContextDetail(contextId: string) {
  const contextStore = useContextStore()
  const tagStore = useTagStore()
  const { currentItem: currentContext, loading } = storeToRefs(contextStore)

  const linkedTasksRef = ref<Task[]>([])
  const tags = computed(() => tagStore.contextTags[contextId] ?? [])
  const linkedTasks = computed(() => linkedTasksRef.value)

  async function load() {
    await Promise.all([
      contextStore.fetchById(contextId),
      tagStore.fetchTagsForContext(contextId),
      fetchAllPages(taskService, { filter: { contextId } }).then((result) => {
        linkedTasksRef.value = result.items
      }),
    ])
  }

  async function update(data: UpdateContext) {
    return contextStore.update(contextId, data)
  }

  async function remove() {
    return contextStore.remove(contextId)
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
    tags,
    linkedTasks,
    loading,
    update,
    remove,
    addTag,
    removeTag,
    reload: load,
  }
}
