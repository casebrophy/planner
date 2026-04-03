import { defineStore } from 'pinia'
import { computed } from 'vue'
import { noteService } from '@/services/noteService'
import { createCRUDStore } from './createCRUDStore'
import type { Note, NewNote, UpdateNote, NoteFilter } from '@/types'

export const useNoteStore = defineStore('note', () => {
  const crud = createCRUDStore<Note, NewNote, UpdateNote, NoteFilter>({
    name: 'note',
    service: noteService,
    defaultOrderBy: 'created_at',
    defaultRowsPerPage: 20,
  })

  const hasActiveFilter = computed(() => {
    const f = crud.filter.value
    return !!(f.contextId || f.source || f.search)
  })

  return {
    ...crud,
    hasActiveFilter,
  }
})
