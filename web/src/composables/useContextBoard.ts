import { onMounted, computed } from 'vue'
import { useContextStore } from '@/stores/contextStore'
import { storeToRefs } from 'pinia'
import { usePolling } from './usePolling'
import type { ContextFilter } from '@/types'

export function useContextBoard() {
  const store = useContextStore()
  const {
    items,
    total,
    loading,
    error,
    filter,
    contextsByStatus,
    activeCount,
    pausedCount,
    closedCount,
  } = storeToRefs(store)

  function setFilter(f: ContextFilter) {
    store.setFilter(f)
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
    contexts: items,
    total,
    loading,
    error,
    filter,
    contextsByStatus,
    activeCount,
    pausedCount,
    closedCount,
    isEmpty,
    setFilter,
    refresh,
  }
}
