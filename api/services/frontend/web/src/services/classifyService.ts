import { request } from './client'

export interface ClassifyAccepted {
  message: string
  unlinkedCount: number
}

export const classifyService = {
  async classify(): Promise<ClassifyAccepted> {
    return request<ClassifyAccepted>('/api/v1/tasks/classify', { method: 'POST' })
  },
}
