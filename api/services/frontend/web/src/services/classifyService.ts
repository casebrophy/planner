import { request } from './client'

export interface ClassifyResult {
  classified: number
  clarificationsCreated: number
}

export const classifyService = {
  async classify(): Promise<ClassifyResult> {
    return request<ClassifyResult>('/api/v1/tasks/classify', { method: 'POST' })
  },
}
