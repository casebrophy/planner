import { request } from './client'

export interface ReceiptExtraction {
  merchant: string
  date: string
  total: number
  tax: number
  subtotal: number
  items: { description: string; amount: number; quantity: number }[]
  notes?: string
}

export const receiptService = {
  extractReceipt(ocrText: string): Promise<ReceiptExtraction> {
    return request<ReceiptExtraction>('/api/v1/transactions/extract-receipt', {
      method: 'POST',
      body: { ocrText },
    })
  },
}
