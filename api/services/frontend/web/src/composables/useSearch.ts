import { ref, computed, watch } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useContextStore } from '@/stores/contextStore'
import { useTagStore } from '@/stores/tagStore'
import { storeToRefs } from 'pinia'

type SearchTab = 'all' | 'tasks' | 'contexts' | 'tags'

export function useSearch() {
  const query = ref('')
  const activeTab = ref<SearchTab>('all')
  const loading = ref(false)
  const hasSearched = ref(false)

  const taskStore = useTaskStore()
  const contextStore = useContextStore()
  const tagStore = useTagStore()

  const { items: tasks } = storeToRefs(taskStore)
  const { items: contexts } = storeToRefs(contextStore)
  const { items: tags } = storeToRefs(tagStore)

  const filteredTasks = computed(() => {
    if (!query.value || query.value.length < 2) return []
    const q = query.value.toLowerCase()
    return tasks.value.filter(
      (t) => t.title.toLowerCase().includes(q) || (t.description && t.description.toLowerCase().includes(q)),
    )
  })

  const filteredContexts = computed(() => {
    if (!query.value || query.value.length < 2) return []
    const q = query.value.toLowerCase()
    return contexts.value.filter(
      (c) => c.title.toLowerCase().includes(q) || (c.description && c.description.toLowerCase().includes(q)),
    )
  })

  const filteredTags = computed(() => {
    if (!query.value || query.value.length < 2) return []
    const q = query.value.toLowerCase()
    return tags.value.filter((t) => t.name.toLowerCase().includes(q))
  })

  const totalResults = computed(() => filteredTasks.value.length + filteredContexts.value.length + filteredTags.value.length)

  async function search() {
    loading.value = true
    try {
      await Promise.all([taskStore.fetchList(true), contextStore.fetchList(true), tagStore.fetchList(true)])
      hasSearched.value = true
    } finally {
      loading.value = false
    }
  }

  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  watch(query, (val) => {
    if (debounceTimer) clearTimeout(debounceTimer)
    if (val.length >= 2) {
      debounceTimer = setTimeout(() => {
        search()
      }, 300)
    } else {
      hasSearched.value = false
    }
  })

  function setTab(tab: SearchTab) {
    activeTab.value = tab
  }

  return {
    query,
    activeTab,
    loading,
    hasSearched,
    filteredTasks,
    filteredContexts,
    filteredTags,
    totalResults,
    search,
    setTab,
  }
}
