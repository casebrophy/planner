import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useToastStore } from './toastStore'
import { clarificationService } from '@/services/clarificationService'
import type { ClarificationItem } from '@/types'

export const useClarificationStore = defineStore('clarification', () => {
  const items = ref<ClarificationItem[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const currentIndex = ref(0)
  const pendingCount = ref(0)

  const toast = useToastStore()

  const currentItem = computed(() => items.value[currentIndex.value] ?? null)
  const hasNext = computed(() => currentIndex.value < items.value.length - 1)
  const isEmpty = computed(() => !loading.value && items.value.length === 0)
  const progress = computed(() => ({
    current: currentIndex.value + 1,
    total: items.value.length,
  }))

  async function fetchQueue(_force = false) {
    loading.value = true
    error.value = null
    try {
      const result = await clarificationService.queryQueue({ status: 'pending', rows: 50 })
      items.value = result.items
      total.value = result.total
      currentIndex.value = 0
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch queue'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function fetchPendingCount() {
    try {
      pendingCount.value = await clarificationService.countPending()
    } catch {
      // Silent fail for badge count
    }
  }

  async function resolve(id: string, answer: Record<string, unknown>) {
    try {
      await clarificationService.resolve(id, answer)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success('Resolved')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to resolve'
      toast.error(msg)
      throw e
    }
  }

  async function snooze(id: string, hours: number = 24) {
    try {
      await clarificationService.snooze(id, hours)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success(`Snoozed for ${hours}h`)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to snooze'
      toast.error(msg)
      throw e
    }
  }

  async function dismiss(id: string) {
    try {
      await clarificationService.dismiss(id)
      removeAndAdvance(id)
      pendingCount.value = Math.max(0, pendingCount.value - 1)
      toast.success('Dismissed')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to dismiss'
      toast.error(msg)
      throw e
    }
  }

  function removeAndAdvance(id: string) {
    const idx = items.value.findIndex((i) => i.id === id)
    if (idx !== -1) {
      items.value.splice(idx, 1)
      total.value--
      // Keep currentIndex valid
      if (currentIndex.value >= items.value.length) {
        currentIndex.value = Math.max(0, items.value.length - 1)
      }
    }
  }

  function goTo(index: number) {
    if (index >= 0 && index < items.value.length) {
      currentIndex.value = index
    }
  }

  return {
    items,
    total,
    loading,
    error,
    currentIndex,
    pendingCount,
    currentItem,
    hasNext,
    isEmpty,
    progress,
    fetchQueue,
    fetchPendingCount,
    resolve,
    snooze,
    dismiss,
    goTo,
  }
})
