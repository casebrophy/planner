// web/src/stores/createCRUDStore.ts
import { ref } from 'vue'
import { useToastStore } from './toastStore'
import type { CRUDService } from '@/services/createCRUDService'

const CACHE_TTL = 5 * 60 * 1000

export interface CRUDStoreConfig<T, TNew, TUpdate, TFilter> {
  name: string
  service: CRUDService<T, TNew, TUpdate, TFilter>
  defaultOrderBy?: string
  defaultRowsPerPage?: number
}

export function createCRUDStore<
  T extends { id: string },
  TNew,
  TUpdate,
  TFilter,
>(config: CRUDStoreConfig<T, TNew, TUpdate, TFilter>) {
  const { name, service, defaultOrderBy = 'created_at', defaultRowsPerPage = 20 } = config

  const items = ref<T[]>([]) as { value: T[] }
  const total = ref(0)
  const page = ref(1)
  const rowsPerPage = ref(defaultRowsPerPage)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const lastFetchedAt = ref<Record<string, number>>({})
  const filter = ref<TFilter>({} as TFilter)
  const orderBy = ref(defaultOrderBy)
  const currentItem = ref<T | null>(null) as { value: T | null }

  const toast = useToastStore()

  function cacheKey(): string {
    return JSON.stringify({ filter: filter.value, orderBy: orderBy.value, page: page.value })
  }

  function isCacheValid(): boolean {
    const key = cacheKey()
    const ts = lastFetchedAt.value[key]
    return ts !== undefined && Date.now() - ts < CACHE_TTL
  }

  async function fetchList(force = false) {
    if (!force && isCacheValid()) return
    loading.value = true
    error.value = null
    try {
      const result = await service.list({
        page: page.value,
        rows: rowsPerPage.value,
        orderBy: orderBy.value,
        filter: filter.value,
      })
      items.value = result.items
      total.value = result.total
      lastFetchedAt.value[cacheKey()] = Date.now()
    } catch (e) {
      error.value = e instanceof Error ? e.message : `Failed to fetch ${name}s`
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchById(id: string) {
    loading.value = true
    error.value = null
    try {
      currentItem.value = await service.getById(id)
    } catch (e) {
      error.value = e instanceof Error ? e.message : `Failed to fetch ${name}`
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function create(item: TNew) {
    try {
      const created = await service.create(item)
      items.value.unshift(created)
      total.value++
      toast.success(`${capitalize(name)} created`)
      return created
    } catch (e) {
      const msg = e instanceof Error ? e.message : `Failed to create ${name}`
      toast.error(msg)
      throw e
    }
  }

  async function update(id: string, data: TUpdate) {
    const idx = items.value.findIndex((item) => item.id === id)
    const backup = idx !== -1 ? { ...items.value[idx]! } : null
    const currentBackup = currentItem.value?.id === id ? { ...currentItem.value } : null

    // Optimistic update
    if (idx !== -1) {
      items.value[idx] = { ...items.value[idx]!, ...stripUndefined(data) }
    }
    if (currentItem.value?.id === id) {
      currentItem.value = { ...currentItem.value, ...stripUndefined(data) }
    }

    try {
      const updated = await service.update(id, data)
      if (idx !== -1) items.value[idx] = updated
      if (currentItem.value?.id === id) currentItem.value = updated
      toast.success(`${capitalize(name)} updated`)
      return updated
    } catch (e) {
      // Rollback
      if (idx !== -1 && backup) items.value[idx] = backup
      if (currentBackup) currentItem.value = currentBackup
      const msg = e instanceof Error ? e.message : `Failed to update ${name}`
      toast.error(msg)
      throw e
    }
  }

  async function remove(id: string) {
    const idx = items.value.findIndex((item) => item.id === id)
    const backup = idx !== -1 ? items.value[idx]! : null

    // Optimistic remove
    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
    }

    try {
      await service.delete(id)
      if (currentItem.value?.id === id) currentItem.value = null
      toast.success(`${capitalize(name)} deleted`)
    } catch (e) {
      // Rollback
      if (backup && idx !== -1) {
        items.value.splice(idx, 0, backup)
        total.value++
      }
      const msg = e instanceof Error ? e.message : `Failed to delete ${name}`
      toast.error(msg)
      throw e
    }
  }

  function setFilter(f: TFilter) {
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
    currentItem,
    lastFetchedAt,
    fetchList,
    fetchById,
    create,
    update,
    remove,
    setFilter,
    setPage,
    setOrder,
  }
}

function stripUndefined(obj: unknown): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, value] of Object.entries(obj as Record<string, unknown>)) {
    if (value !== undefined) result[key] = value
  }
  return result
}

function capitalize(s: string): string {
  return s.charAt(0).toUpperCase() + s.slice(1)
}
