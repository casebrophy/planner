import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { contextService } from '@/services/contextService'
import { createCRUDStore } from './createCRUDStore'
import { useToastStore } from './toastStore'
import type { Context, NewContext, UpdateContext, ContextFilter, ContextEvent, NewEvent } from '@/types'
import { ContextStatus } from '@/types'

export const useContextStore = defineStore('context', () => {
  const crud = createCRUDStore<Context, NewContext, UpdateContext, ContextFilter>({
    name: 'context',
    service: contextService,
    defaultOrderBy: 'last_event',
    defaultRowsPerPage: 50,
  })

  const events = ref<ContextEvent[]>([])
  const eventsTotal = ref(0)

  const toast = useToastStore()

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

  const activeCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Active).length)
  const pausedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Paused).length)
  const closedCount = computed(() => crud.items.value.filter((c) => c.status === ContextStatus.Closed).length)

  async function fetchEvents(contextId: string, pg = 1) {
    try {
      const result = await contextService.listEvents(contextId, { page: pg, rows: 50 })
      events.value = result.items
      eventsTotal.value = result.total
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to fetch events'
      toast.error(msg)
    }
  }

  async function addEvent(contextId: string, event: NewEvent) {
    try {
      const created = await contextService.addEvent(contextId, event)
      events.value.unshift(created)
      eventsTotal.value++
      toast.success('Event added')
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to add event'
      toast.error(msg)
      throw e
    }
  }

  return {
    ...crud,
    events,
    eventsTotal,
    contextsByStatus,
    activeCount,
    pausedCount,
    closedCount,
    fetchEvents,
    addEvent,
  }
})
