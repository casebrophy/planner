import { computed, type Ref } from 'vue'

export function usePagination(page: Ref<number>, rowsPerPage: Ref<number>, total: Ref<number>) {
  const totalPages = computed(() => Math.max(1, Math.ceil(total.value / rowsPerPage.value)))
  const hasNextPage = computed(() => page.value < totalPages.value)
  const hasPrevPage = computed(() => page.value > 1)

  function nextPage() {
    if (hasNextPage.value) page.value++
  }

  function prevPage() {
    if (hasPrevPage.value) page.value--
  }

  function goToPage(p: number) {
    if (p >= 1 && p <= totalPages.value) page.value = p
  }

  return { totalPages, hasNextPage, hasPrevPage, nextPage, prevPage, goToPage }
}
