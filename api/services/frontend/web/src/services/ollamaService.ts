import { request } from './client'
import type { OllamaStatus, PullResult } from '@/types/ollama'

export const ollamaService = {
  async getStatus(): Promise<OllamaStatus> {
    return request<OllamaStatus>('/api/v1/ollama/status')
  },

  async pullModel(model: string): Promise<PullResult> {
    return request<PullResult>(`/api/v1/ollama/pull/${encodeURIComponent(model)}`, {
      method: 'POST',
    })
  },
}
