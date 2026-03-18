import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { contextService } from '@/services/contextService'
import { useToastStore } from './toastStore'
import type { Context, NewContext, UpdateContext, ContextFilter, ContextEvent, NewEvent } from '@/types'
import { ContextStatus } from '@/types'

const CACHE_TTL = 5 * 60 * 1000

export const useContextStore = defineStore('context', () => {
  const items = ref<Context[]>([])
  const total = ref(0)
  const page = ref(1)
  const rowsPerPage = ref(50)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastFetchedAt = ref<Record<string, number>>({})
  const filter = ref<ContextFilter>({})
  const orderBy = ref('last_event')
  const currentContext = ref<Context | null>(null)
  const events = ref<ContextEvent[]>([])
  const eventsTotal = ref(0)

  const toast = useToastStore()

  const contextsByStatus = computed(() => {
    const groups: Record<string, Context[]> = {
      [ContextStatus.Active]: [],
      [ContextStatus.Paused]: [],
      [ContextStatus.Closed]: [],
    }
    for (const ctx of items.value) {
      groups[ctx.status]?.push(ctx)
    }
    return groups
  })

  const activeCount = computed(() => items.value.filter((c) => c.status === ContextStatus.Active).length)
  const pausedCount = computed(() => items.value.filter((c) => c.status === ContextStatus.Paused).length)
  const closedCount = computed(() => items.value.filter((c) => c.status === ContextStatus.Closed).length)

  function cacheKey(): string {
    return JSON.stringify({ filter: filter.value, orderBy: orderBy.value, page: page.value })
  }

  function isCacheValid(): boolean {
    const key = cacheKey()
    const ts = lastFetchedAt.value[key]
    return ts !== undefined && Date.now() - ts < CACHE_TTL
  }

  async function fetchContexts(force = false) {
    if (!force && isCacheValid()) return
    loading.value = true
    error.value = null
    try {
      const result = await contextService.list({
        page: page.value,
        rows: rowsPerPage.value,
        orderBy: orderBy.value,
        filter: filter.value,
      })
      items.value = result.items
      total.value = result.total
      lastFetchedAt.value[cacheKey()] = Date.now()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch contexts'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchContext(id: string) {
    loading.value = true
    error.value = null
    try {
      currentContext.value = await contextService.getById(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch context'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function createContext(ctx: NewContext) {
    try {
      const created = await contextService.create(ctx)
      items.value.unshift(created)
      total.value++
      toast.success('Context created')
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to create context'
      toast.error(msg)
      throw e
    }
  }

  async function updateContext(id: string, update: UpdateContext) {
    const idx = items.value.findIndex((c) => c.id === id)
    const backup = idx !== -1 ? { ...items.value[idx]! } : null

    if (idx !== -1) {
      items.value[idx] = { ...items.value[idx]!, ...stripUndefined(update) }
    }
    if (currentContext.value?.id === id) {
      currentContext.value = { ...currentContext.value, ...stripUndefined(update) }
    }

    try {
      const updated = await contextService.update(id, update)
      if (idx !== -1) items.value[idx] = updated
      if (currentContext.value?.id === id) currentContext.value = updated
      toast.success('Context updated')
      return updated
    } catch (e) {
      if (idx !== -1 && backup) items.value[idx] = backup
      if (currentContext.value?.id === id && backup) currentContext.value = backup
      const msg = e instanceof Error ? e.message : 'Failed to update context'
      toast.error(msg)
      throw e
    }
  }

  async function deleteContext(id: string) {
    const idx = items.value.findIndex((c) => c.id === id)
    const backup = idx !== -1 ? items.value[idx]! : null

    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
    }

    try {
      await contextService.delete(id)
      if (currentContext.value?.id === id) currentContext.value = null
      toast.success('Context deleted')
    } catch (e) {
      if (backup && idx !== -1) {
        items.value.splice(idx, 0, backup)
        total.value++
      }
      const msg = e instanceof Error ? e.message : 'Failed to delete context'
      toast.error(msg)
      throw e
    }
  }

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

  function setFilter(f: ContextFilter) {
    filter.value = f
    page.value = 1
  }

  function setPage(p: number) {
    page.value = p
  }

  function setOrder(o: string) {
    orderBy.value = o
    page.value = 1
  }

  return {
    items,
    total,
    page,
    rowsPerPage,
    loading,
    error,
    filter,
    orderBy,
    currentContext,
    events,
    eventsTotal,
    lastFetchedAt,
    contextsByStatus,
    activeCount,
    pausedCount,
    closedCount,
    fetchContexts,
    fetchContext,
    createContext,
    updateContext,
    deleteContext,
    fetchEvents,
    addEvent,
    setFilter,
    setPage,
    setOrder,
  }
})

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function stripUndefined(obj: any): any {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj)) {
    if (value !== undefined) result[key] = value
  }
  return result
}
