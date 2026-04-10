export interface ModelInfo {
  name: string
  available: boolean
}

export interface OllamaStatus {
  reachable: boolean
  extractModel: ModelInfo
  embedModel: ModelInfo
  allModels: string[]
}

export interface PullResult {
  status: string
}
