import { ref, onMounted, onUnmounted } from 'vue'
import { request } from '@/services/client'

export interface ContainerInfo {
  name: string
  state: string
  status: string
  health: string
  image: string
  ports: string
}

export interface TimerInfo {
  name: string
  active: boolean
  lastRun: string
  nextRun: string
  result: string
}

export interface ClaudeInstance {
  pid: string
  command: string
  cpu: string
  memory: string
  elapsed: string
}

export function useServerMonitor() {
  const containers = ref<ContainerInfo[]>([])
  const timers = ref<TimerInfo[]>([])
  const claudeInstances = ref<ClaudeInstance[]>([])
  const logs = ref<string>('')
  const logService = ref<string>('backend')
  const loading = ref(false)
  const error = ref<string>('')
  const available = ref(true)

  let pollInterval: ReturnType<typeof setInterval> | null = null

  async function fetchContainers() {
    try {
      containers.value = await request<ContainerInfo[]>('/api/v1/server/containers')
    } catch (e: any) {
      if (e?.message?.includes('sidecar')) {
        available.value = false
      }
      error.value = e?.message || 'Failed to fetch containers'
    }
  }

  async function fetchTimers() {
    try {
      timers.value = await request<TimerInfo[]>('/api/v1/server/timers')
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch timers'
    }
  }

  async function fetchClaude() {
    try {
      claudeInstances.value = await request<ClaudeInstance[]>('/api/v1/server/claude')
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch Claude instances'
    }
  }

  async function fetchLogs(service?: string) {
    if (service) logService.value = service
    try {
      const resp = await request<{ logs: string }>(`/api/v1/server/logs/${logService.value}?lines=100`)
      logs.value = resp.logs
    } catch (e: any) {
      error.value = e?.message || 'Failed to fetch logs'
    }
  }

  async function refresh() {
    loading.value = true
    error.value = ''
    await Promise.all([fetchContainers(), fetchTimers(), fetchClaude()])
    loading.value = false
  }

  onMounted(() => {
    refresh()
    fetchLogs()
    pollInterval = setInterval(refresh, 30000)
  })

  onUnmounted(() => {
    if (pollInterval) clearInterval(pollInterval)
  })

  return {
    containers,
    timers,
    claudeInstances,
    logs,
    logService,
    loading,
    error,
    available,
    refresh,
    fetchLogs,
  }
}
