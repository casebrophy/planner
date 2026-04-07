import { ref } from 'vue'
import { defineStore } from 'pinia'
import { entityLinkService } from '@/services/entityLinkService'
import { useToastStore } from './toastStore'
import type { EntityLink, NewEntityLink } from '@/types'

export const useEntityLinkStore = defineStore('entityLink', () => {
  // Keyed by `${entityType}:${entityId}` → EntityLink[]
  const linksByEntity = ref<Record<string, EntityLink[]>>({})
  const loading = ref(false)

  function cacheKey(entityType: string, entityId: string): string {
    return `${entityType}:${entityId}`
  }

  async function fetchLinks(entityType: string, entityId: string, force = false): Promise<void> {
    const key = cacheKey(entityType, entityId)
    if (!force && linksByEntity.value[key] !== undefined) return

    loading.value = true
    try {
      linksByEntity.value[key] = await entityLinkService.listByEntity(entityType, entityId)
    } catch (e) {
      useToastStore().error('Failed to load related items')
    } finally {
      loading.value = false
    }
  }

  function getLinks(entityType: string, entityId: string): EntityLink[] {
    return linksByEntity.value[cacheKey(entityType, entityId)] ?? []
  }

  async function createLink(link: NewEntityLink): Promise<EntityLink | null> {
    try {
      const created = await entityLinkService.create(link)
      // Invalidate cache for both sides
      delete linksByEntity.value[cacheKey(link.sourceType, link.sourceId)]
      delete linksByEntity.value[cacheKey(link.targetType, link.targetId)]
      return created
    } catch (e) {
      useToastStore().error('Failed to create link')
      return null
    }
  }

  async function deleteLink(link: EntityLink): Promise<void> {
    try {
      await entityLinkService.delete(link.id)
      // Invalidate cache for both sides
      delete linksByEntity.value[cacheKey(link.sourceType, link.sourceId)]
      delete linksByEntity.value[cacheKey(link.targetType, link.targetId)]
    } catch (e) {
      useToastStore().error('Failed to remove link')
    }
  }

  return { linksByEntity, loading, fetchLinks, getLinks, createLink, deleteLink }
})
