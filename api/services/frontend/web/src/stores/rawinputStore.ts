import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useToastStore } from './toastStore'
import { rawinputService } from '@/services/rawinputService'
import type { RawInput } from '@/types/rawinput'

export const useRawInputStore = defineStore('rawinput', () => {
  const items = ref<RawInput[]>([])
  const total = ref(0)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const page = ref(1)
  const rowsPerPage = ref(25)
  const statusFilter = ref<string | undefined>(undefined)

  const toast = useToastStore()

  const totalPages = computed(() => Math.ceil(total.value / rowsPerPage.value))
  const failedCount = computed(() => items.value.filter((i) => i.status === 'failed').length)

  async function fetchList(force = false) {
    if (loading.value && !force) return
    loading.value = true
    error.value = null
    try {
      const result = await rawinputService.list({
        page: page.value,
        rows: rowsPerPage.value,
        status: statusFilter.value,
        orderBy: 'created_at',
      })
      items.value = result.items
      total.value = result.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to fetch raw inputs'
      toast.error(error.value)
    } finally {
      loading.value = false
    }
  }

  async function reprocess(id: string) {
    try {
      await rawinputService.reprocess(id)
      toast.success('Requeued for processing')
      await fetchList(true)
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to reprocess'
      toast.error(msg)
      throw e
    }
  }

  async function setStatusFilter(status: string | undefined) {
    statusFilter.value = status
    page.value = 1
    await fetchList(true)
  }

  function setPage(newPage: number) {
    page.value = newPage
    fetchList(true)
  }

  return {
    items,
    total,
    loading,
    error,
    page,
    rowsPerPage,
    statusFilter,
    totalPages,
    failedCount,
    fetchList,
    reprocess,
    setStatusFilter,
    setPage,
  }
})
