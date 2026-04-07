import { onMounted, computed, type Ref } from 'vue'
import { useNoteStore } from '@/stores/noteStore'
import { storeToRefs } from 'pinia'
import type { NewNote, UpdateNote } from '@/types'

export function useTaskNotes(taskId: string | Ref<string>) {
  const noteStore = useNoteStore()
  const { items, loading } = storeToRefs(noteStore)

  // Resolve taskId whether it's a string or Ref
  const resolvedTaskId = computed(() =>
    typeof taskId === 'string' ? taskId : taskId.value,
  )

  // Items are already filtered server-side via setFilter; expose them as-is
  const notes = computed(() => items.value)

  async function load() {
    if (!resolvedTaskId.value) return
    noteStore.setFilter({ taskId: resolvedTaskId.value })
    await noteStore.fetchList(true)
  }

  async function addNote(data: NewNote) {
    return noteStore.create(data)
  }

  async function updateNote(noteId: string, data: UpdateNote) {
    return noteStore.update(noteId, data)
  }

  async function deleteNote(noteId: string) {
    return noteStore.remove(noteId)
  }

  onMounted(load)

  return {
    notes,
    loading,
    addNote,
    updateNote,
    deleteNote,
    reload: load,
  }
}
