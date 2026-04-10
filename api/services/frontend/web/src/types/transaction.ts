export interface Transaction {
  id: string
  rawInputId?: string
  source: string
  date: string
  description: string
  cleanName?: string
  amount: number
  category?: string
  contextId?: string
  notes?: string
  reviewed: boolean
  createdAt: string
}

export interface UpdateTransaction {
  cleanName?: string
  category?: string
  contextId?: string
  notes?: string
  reviewed?: boolean
}

export interface TransactionFilter {
  contextId?: string
  source?: string
  reviewed?: boolean
  category?: string
}

export interface ImportResult {
  total: number
  imported: number
  skipped: number
}

export interface EnrichmentStatus {
  active: number
  pending: number
  done: number
  failed: number
  enabled: boolean
}
