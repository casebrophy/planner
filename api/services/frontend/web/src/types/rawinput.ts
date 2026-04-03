export interface RawInput {
  id: string
  sourceType: string
  status: string
  rawContent: string
  processedAt?: string
  error?: string
  retryCount: number
  nextRetryAt?: string
  maxRetries: number
  createdAt: string
}
