import { createCRUDService } from './createCRUDService'
import { request } from './client'
import type { Note, NewNote, UpdateNote, NoteFilter, Task } from '@/types'

const noteCrud = createCRUDService<Note, NewNote, UpdateNote, NoteFilter>({
  basePath: '/api/v1/notes',
  mapFilter: (f) => ({
    context_id: f.contextId,
    task_id: f.taskId,
    source: f.source,
    search: f.search,
  }),
})

export const noteService = {
  ...noteCrud,
  convertNoteToTask: (noteId: string): Promise<Task> =>
    request<Task>(`/api/v1/notes/${noteId}/convert-to-task`, { method: 'POST' }),
}
