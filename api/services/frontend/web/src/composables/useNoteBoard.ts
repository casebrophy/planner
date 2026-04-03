import { onMounted, computed, toRef } from 'vue'
import { useNoteStore } from '@/stores/noteStore'
import { storeToRefs } from 'pinia'
import { usePagination } from './usePagination'
import { usePolling } from './usePolling'
import type { NoteFilter } from '@/types'

export function useNoteBoard() {
  const store = useNoteStore()
  const { items, total, page, rowsPerPage, loading, error, orderBy, hasActiveFilter } =
    storeToRefs(store)

  const pagination = usePagination(page, rowsPerPage, total)

  function setFilter(f: NoteFilter) {
    store.setFilter(f)
    store.fetchList(true)
  }

  function setOrder(o: string) {
    store.setOrder(o)
    store.fetchList(true)
  }

  function setPage(p: number) {
    store.setPage(p)
    store.fetchList(true)
  }

  function refresh() {
    store.fetchList(true)
  }

  onMounted(() => {
    store.fetchList()
  })

  usePolling(() => store.fetchList(true))

  const isEmpty = computed(() => !loading.value && items.value.length === 0)

  return {
    notes: items,
    total,
    page,
    rowsPerPage,
    loading,
    error,
    filter: toRef(store, 'filter'),
    orderBy,
    hasActiveFilter,
    pagination,
    isEmpty,
    setFilter,
    setOrder,
    setPage,
    refresh,
  }
}
