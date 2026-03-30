import { request } from './client'

export interface ThreadEntry {
  id: string
  subjectType: string
  subjectId: string
  kind: string
  content: string
  source: string
  metadata?: Record<string, unknown>
  sourceId?: string
  sentiment?: string
  requiresAction: boolean
  createdAt: string
}

interface ThreadQueryResponse {
  items: ThreadEntry[]
  total: number
  page: number
  rowsPerPage: number
}

export const threadService = {
  async queryBySubject(subjectType: string, subjectId: string): Promise<ThreadEntry[]> {
    const res = await request<ThreadQueryResponse>(
      `/api/v1/threads/${subjectType}/${subjectId}`,
    )
    return res.items
  },
}
