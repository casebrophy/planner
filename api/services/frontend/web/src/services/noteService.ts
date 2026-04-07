import { createCRUDService } from './createCRUDService'
import type { Note, NewNote, UpdateNote, NoteFilter } from '@/types'

export const noteService = createCRUDService<Note, NewNote, UpdateNote, NoteFilter>({
  basePath: '/api/v1/notes',
  mapFilter: (f) => ({
    context_id: f.contextId,
    task_id: f.taskId,
    source: f.source,
    search: f.search,
  }),
})
