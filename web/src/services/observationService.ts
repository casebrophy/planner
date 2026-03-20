import { request } from './client'

export interface Observation {
  id: string
  subjectType: string
  subjectId: string
  kind: string
  data: Record<string, unknown>
  source: string
  confidence: number
  weight: number
  createdAt: string
}

interface ObservationQueryResponse {
  items: Observation[]
  total: number
  page: number
  rowsPerPage: number
}

export const observationService = {
  async queryBySubject(subjectType: string, subjectId: string): Promise<Observation[]> {
    const res = await request<ObservationQueryResponse>(
      `/api/v1/observations/${subjectType}/${subjectId}`,
    )
    return res.items
  },
}
