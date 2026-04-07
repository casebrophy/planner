import { request } from './client'
import type { EntityLink, NewEntityLink } from '@/types'

export const entityLinkService = {
  async listByEntity(entityType: string, entityId: string): Promise<EntityLink[]> {
    const result = await request<{ items: EntityLink[]; total: number }>('/api/v1/entity-links', {
      params: { entity_type: entityType, entity_id: entityId },
    })
    return result.items
  },

  async create(link: NewEntityLink): Promise<EntityLink> {
    return request<EntityLink>('/api/v1/entity-links', { method: 'POST', body: link })
  },

  async delete(id: string): Promise<void> {
    return request<void>(`/api/v1/entity-links/${id}`, { method: 'DELETE' })
  },
}
