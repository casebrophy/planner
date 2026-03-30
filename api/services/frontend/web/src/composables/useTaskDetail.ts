import { onMounted, computed } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useTagStore } from '@/stores/tagStore'
import { storeToRefs } from 'pinia'
import type { UpdateTask } from '@/types'

export function useTaskDetail(taskId: string) {
  const taskStore = useTaskStore()
  const tagStore = useTagStore()
  const { currentItem: currentTask, loading } = storeToRefs(taskStore)

  const tags = computed(() => tagStore.taskTags[taskId] ?? [])

  async function load() {
    await Promise.all([taskStore.fetchById(taskId), tagStore.fetchTagsForTask(taskId)])
  }

  async function update(data: UpdateTask) {
    return taskStore.update(taskId, data)
  }

  async function remove() {
    return taskStore.remove(taskId)
  }

  async function addTag(tagId: string) {
    return tagStore.addTagToTask(taskId, tagId)
  }

  async function removeTag(tagId: string) {
    return tagStore.removeTagFromTask(taskId, tagId)
  }

  onMounted(load)

  return {
    task: currentTask,
    tags,
    loading,
    update,
    remove,
    addTag,
    removeTag,
    reload: load,
  }
}
