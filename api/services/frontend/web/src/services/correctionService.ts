import { request } from './client'

interface CorrectionResult {
  id: string
  type: string
}

export const correctionService = {
  async correct(itemId: string, itemType: string, newType: string): Promise<CorrectionResult> {
    return request<CorrectionResult>('/api/v1/corrections', {
      method: 'POST',
      body: {
        item_id: itemId,
        item_type: itemType,
        new_type: newType,
      },
    })
  },
}
