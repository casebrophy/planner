export interface Note {
  id: string
  contextId?: string
  content: string
  source: string
  rawInputId?: string
  createdAt: string
  updatedAt: string
}

export interface NewNote {
  contextId?: string
  content: string
  source: string
}

export interface UpdateNote {
  contextId?: string
  content?: string
  source?: string
}

export interface NoteFilter {
  contextId?: string
  taskId?: string
  source?: string
  search?: string
}
