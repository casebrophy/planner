import { ref, onMounted, onUnmounted } from 'vue'
import { request } from '@/services/client'

function errMsg(e: unknown): string | undefined {
  return e instanceof Error ? e.message : undefined
}

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

export interface InferenceStatus {
  session_id: string
  created_at: string
  age_seconds: number
  total_requests: number
  latest_input_tokens: number
  context_max: number
  context_usage_pct: number
  token_growth: number[]
  avg_duration_ms: number
  duration_trend: number[]
  requests_since_rotation: number
}

export interface SessionSummary {
  session_id: string
  created_at: string
  ended_at: string
  total_requests: number
  end_reason: string
  peak_input_tokens: number
}

export interface InferenceHistory {
  sessions: SessionSummary[]
}

export interface InferenceTools {
  tool_frequency: Record<string, number>
  avg_calls_per_request: number
}

export interface SidecarLogEntry {
  id: string
  timestamp: string
  duration_ms: number
  agent_model: string
  prompt_prefix: string
  input_tokens: number
  output_tokens: number
  session_id: string
  success: boolean
  error?: string
}

export function useServerMonitor() {
  const containers = ref<ContainerInfo[]>([])
  const timers = ref<TimerInfo[]>([])
  const claudeInstances = ref<ClaudeInstance[]>([])
  const logs = ref<string>('')
  const sidecarLogs = ref<SidecarLogEntry[]>([])
  const logService = ref<string>('backend')
  const inferenceStatus = ref<InferenceStatus | null>(null)
  const inferenceHistory = ref<SessionSummary[]>([])
  const inferenceTools = ref<InferenceTools | null>(null)
  const loading = ref(false)
  const error = ref<string>('')
  const available = ref(true)

  let pollInterval: ReturnType<typeof setInterval> | null = null

  async function fetchContainers() {
    try {
      containers.value = await request<ContainerInfo[]>('/api/v1/server/containers')
    } catch (e: unknown) {
      if (errMsg(e)?.includes('sidecar')) {
        available.value = false
      }
      error.value = errMsg(e) || 'Failed to fetch containers'
    }
  }

  async function fetchTimers() {
    try {
      timers.value = await request<TimerInfo[]>('/api/v1/server/timers')
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch timers'
    }
  }

  async function fetchClaude() {
    try {
      claudeInstances.value = await request<ClaudeInstance[]>('/api/v1/server/claude')
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch Claude instances'
    }
  }

  async function fetchLogs(service?: string) {
    if (service) logService.value = service
    try {
      if (logService.value === 'sidecar') {
        const entries = await request<SidecarLogEntry[]>('/api/v1/server/logs/sidecar?limit=50')
        sidecarLogs.value = entries
      } else {
        const resp = await request<{ logs: string }>(`/api/v1/server/logs/${logService.value}?lines=100`)
        logs.value = resp.logs
      }
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch logs'
    }
  }

  async function fetchInferenceStatus() {
    try {
      inferenceStatus.value = await request<InferenceStatus>('/api/v1/server/inference/status')
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch inference status'
    }
  }

  async function fetchInferenceHistory() {
    try {
      const resp = await request<InferenceHistory>('/api/v1/server/inference/history')
      inferenceHistory.value = resp.sessions
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch inference history'
    }
  }

  async function fetchInferenceTools() {
    try {
      inferenceTools.value = await request<InferenceTools>('/api/v1/server/inference/tools')
    } catch (e: unknown) {
      error.value = errMsg(e) || 'Failed to fetch inference tools'
    }
  }

  async function refreshInference() {
    await Promise.all([fetchInferenceStatus(), fetchInferenceHistory(), fetchInferenceTools()])
  }

  async function refresh() {
    loading.value = true
    error.value = ''
    await Promise.all([fetchContainers(), fetchTimers(), fetchClaude(), refreshInference()])
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
    sidecarLogs,
    logService,
    inferenceStatus,
    inferenceHistory,
    inferenceTools,
    loading,
    error,
    available,
    refresh,
    fetchLogs,
    refreshInference,
  }
}
