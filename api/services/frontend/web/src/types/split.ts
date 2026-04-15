export interface Split {
  id: string
  transactionId: string
  partyName: string
  amount: number
  venmoHandle?: string
  settled: boolean
  createdAt: string
  updatedAt: string
}

export interface NewSplit {
  transactionId: string
  partyName: string
  amount: number
  venmoHandle?: string
}

export interface UpdateSplit {
  partyName?: string
  amount?: number
  venmoHandle?: string
  settled?: boolean
}
