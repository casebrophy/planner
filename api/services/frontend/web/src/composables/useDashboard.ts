import { onMounted, computed, ref } from 'vue'
import { useTaskStore } from '@/stores/taskStore'
import { useContextStore } from '@/stores/contextStore'
import { storeToRefs } from 'pinia'
import { usePolling } from './usePolling'
import { TaskStatus, ContextStatus } from '@/types'

export function useDashboard() {
  const taskStore = useTaskStore()
  const contextStore = useContextStore()
  const { items: tasks } = storeToRefs(taskStore)
  const { items: contexts } = storeToRefs(contextStore)
  const loading = ref(false)

  const taskCounts = computed(() => ({
    total: tasks.value.length,
    todo: tasks.value.filter((t) => t.status === TaskStatus.Todo).length,
    inProgress: tasks.value.filter((t) => t.status === TaskStatus.InProgress).length,
    done: tasks.value.filter((t) => t.status === TaskStatus.Done).length,
    overdue: tasks.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < new Date() &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Cancelled,
    ).length,
  }))

  const contextCounts = computed(() => ({
    total: contexts.value.length,
    active: contexts.value.filter((c) => c.status === ContextStatus.Active).length,
    paused: contexts.value.filter((c) => c.status === ContextStatus.Paused).length,
    closed: contexts.value.filter((c) => c.status === ContextStatus.Closed).length,
  }))

  const recentTasks = computed(() =>
    [...tasks.value].sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()).slice(0, 5),
  )

  const overdueTasks = computed(() =>
    tasks.value.filter(
      (t) =>
        t.dueDate &&
        new Date(t.dueDate) < new Date() &&
        t.status !== TaskStatus.Done &&
        t.status !== TaskStatus.Cancelled,
    ),
  )

  const activeContexts = computed(() =>
    contexts.value.filter((c) => c.status === ContextStatus.Active),
  )

  async function load() {
    loading.value = true
    try {
      await Promise.all([taskStore.fetchList(true), contextStore.fetchList(true)])
    } finally {
      loading.value = false
    }
  }

  onMounted(load)
  usePolling(load)

  return {
    loading,
    taskCounts,
    contextCounts,
    recentTasks,
    overdueTasks,
    activeContexts,
    refresh: load,
  }
}
