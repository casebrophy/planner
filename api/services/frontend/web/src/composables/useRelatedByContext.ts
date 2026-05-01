import { ref, watch, type Ref } from 'vue'
import { taskService } from '@/services/taskService'
import { noteService } from '@/services/noteService'
import { fetchAllPages } from '@/services/fetchAllPages'
import type { Task, Note } from '@/types'

export function useRelatedByContext(
  contextId: Ref<string | undefined>,
  entityType: 'task' | 'note',
  entityId: string,
) {
  const tasks = ref<Task[]>([])
  const notes = ref<Note[]>([])
  const loading = ref(false)

  async function fetch() {
    const cid = contextId.value
    if (!cid) {
      tasks.value = []
      notes.value = []
      return
    }

    loading.value = true
    try {
      const [taskResult, noteResult] = await Promise.all([
        fetchAllPages(taskService, { filter: { contextId: cid }, orderBy: 'created_at' }),
        fetchAllPages(noteService, { filter: { contextId: cid }, orderBy: 'created_at' }),
      ])

      tasks.value = taskResult.items.filter(t => !(entityType === 'task' && t.id === entityId))
      notes.value = noteResult.items.filter(n => !(entityType === 'note' && n.id === entityId))
    } finally {
      loading.value = false
    }
  }

  watch(contextId, fetch, { immediate: true })

  return { tasks, notes, loading }
}
