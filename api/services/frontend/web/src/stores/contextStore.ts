import { defineStore } from 'pinia'
import { computed } from 'vue'
import { contextService } from '@/services/contextService'
import { createCRUDStore } from './createCRUDStore'
import type { Context, NewContext, UpdateContext, ContextFilter } from '@/types'
import { ContextStatus, ContextKind } from '@/types'

export const useContextStore = defineStore('context', () => {
  const crud = createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>({
    name: 'context',
    service: contextService,
    defaultOrderBy: 'last_event',
    defaultRowsPerPage: 50,
  })

  const contextsByStatus = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextStatus.Active]: [],
      [ContextStatus.Paused]: [],
      [ContextStatus.Closed]: [],
    }
    for (const ctx of crud.items.value) {
      groups[ctx.status]?.push(ctx)
    }
    return groups
  })

  const contextsByKind = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextKind.Project]: [],
      [ContextKind.Area]: [],
      [ContextKind.List]: [],
    }
    for (const ctx of crud.items.value) {
      const bucket = groups[ctx.kind]
      if (bucket) {
        bucket.push(ctx)
      } else {
        groups[ContextKind.Project]!.push(ctx) // fallback for unknown kind
      }
    }
    return groups
  })

  const contextById = computed(() => (id: string): Context | undefined => {
    return crud.items.value.find((c) => c.id === id)
  })

  const activeCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Active).length)
  const pausedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Paused).length)
  const closedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Closed).length)

  return {
    ...crud,
    contextsByStatus,
    contextsByKind,
    contextById,
    activeCount,
    pausedCount,
    closedCount,
  }
})
