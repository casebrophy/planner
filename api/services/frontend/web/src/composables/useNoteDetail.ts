import { onMounted, computed } from 'vue'
import { useNoteStore } from '@/stores/noteStore'
import { useTagStore } from '@/stores/tagStore'
import { storeToRefs } from 'pinia'
import type { UpdateNote } from '@/types'

export function useNoteDetail(noteId: string) {
  const noteStore = useNoteStore()
  const tagStore = useTagStore()
  const { currentItem: currentNote, loading } = storeToRefs(noteStore)

  const tags = computed(() => tagStore.noteTags[noteId] ?? [])

  async function load() {
    await Promise.all([noteStore.fetchById(noteId), tagStore.fetchTagsForNote(noteId)])
  }

  async function update(data: UpdateNote) {
    return noteStore.update(noteId, data)
  }

  async function remove() {
    return noteStore.remove(noteId)
  }

  async function addTag(tagId: string) {
    return tagStore.addTagToNote(noteId, tagId)
  }

  async function removeTag(tagId: string) {
    return tagStore.removeTagFromNote(noteId, tagId)
  }

  onMounted(load)

  return {
    note: currentNote,
    tags,
    loading,
    update,
    remove,
    addTag,
    removeTag,
    reload: load,
  }
}
