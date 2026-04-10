import { computed, onMounted, ref, watch } from 'vue'
import { storeToRefs } from 'pinia'
import { useDailyPlanStore } from '@/stores/dailyPlanStore'
import { useTaskStore } from '@/stores/taskStore'
import { usePolling } from '@/composables/usePolling'
import type { DailyPlanItem } from '@/types/dailyPlan'
import type { Task } from '@/types'

export function useDailyPlan() {
  const dailyPlanStore = useDailyPlanStore()
  const taskStore = useTaskStore()

  const { plan, loading, generating } = storeToRefs(dailyPlanStore)
  const { items: tasks } = storeToRefs(taskStore)

  // Group items by groupName, sorted by groupPosition then position
  const groupedItems = computed(() => {
    if (!plan.value?.items) return []

    const groups: Map<string, { name: string; position: number; items: DailyPlanItem[] }> = new Map()

    for (const item of plan.value.items) {
      if (item.status === 'dismissed') continue // hide dismissed items

      if (!groups.has(item.groupName)) {
        groups.set(item.groupName, {
          name: item.groupName,
          position: item.groupPosition,
          items: [],
        })
      }
      groups.get(item.groupName)!.items.push(item)
    }

    // Sort groups by position, items within by position
    const sorted = Array.from(groups.values()).sort((a, b) => a.position - b.position)

    for (const group of sorted) {
      group.items.sort((a, b) => (a.userPosition ?? a.position) - (b.userPosition ?? b.position))
    }

    return sorted
  })

  // Build task lookup map — merge new data into existing map so completed
  // tasks (filtered out of the active list) keep their title visible.
  const taskMap = ref<Record<string, Task>>({})
  watch(tasks, (current) => {
    for (const t of current) {
      taskMap.value[t.id] = t
    }
  }, { immediate: true })

  // Stats
  const completedCount = computed(() => plan.value?.items.filter((i) => i.status === 'completed').length ?? 0)
  const totalCount = computed(() => plan.value?.items.filter((i) => i.status !== 'dismissed').length ?? 0)

  async function load() {
    await Promise.all([dailyPlanStore.fetchPlan(), taskStore.fetchList(true)])
  }

  onMounted(load)
  usePolling(load)

  return {
    plan,
    loading,
    generating,
    groupedItems,
    taskMap,
    completedCount,
    totalCount,
    refresh: load,
    regenerate: () => dailyPlanStore.regenerate(),
    completeItem: (id: string) => dailyPlanStore.completeItem(id),
    dismissItem: (id: string, reason: string, note?: string) => dailyPlanStore.dismissItem(id, reason, note),
    reorderItem: (id: string, pos: number) => dailyPlanStore.reorderItem(id, pos),
  }
}
