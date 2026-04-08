export interface StepResult {
  status: string
  detail?: Record<string, unknown>
}

export interface PipelineResult {
  sanitize?: StepResult
  extraction?: StepResult
  contextMatch?: StepResult
  tasks?: StepResult
  events?: StepResult
  notes?: StepResult
}

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
  result?: PipelineResult
  createdAt: string
}
