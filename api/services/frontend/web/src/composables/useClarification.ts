import { onMounted } from 'vue'
import { storeToRefs } from 'pinia'
import { useClarificationStore } from '@/stores/clarificationStore'

export function useClarification() {
  const store = useClarificationStore()
  const { items, total, loading, error, currentItem, isEmpty, progress, pendingCount } =
    storeToRefs(store)

  // Pending count polling is handled globally by AppSidebar (60s interval).
  // This composable only fetches the queue itself.
  onMounted(() => {
    store.fetchQueue()
  })

  return {
    items,
    total,
    loading,
    error,
    currentItem,
    isEmpty,
    progress,
    pendingCount,
    resolve: store.resolve,
    snooze: store.snooze,
    dismiss: store.dismiss,
    refresh: () => store.fetchQueue(true),
  }
}
