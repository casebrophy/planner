<script setup lang="ts">
import { useSettings } from '@/composables/useSettings'
import { useServerMonitor } from '@/composables/useServerMonitor'
import PageHeader from '@/components/layout/PageHeader.vue'
import { computed, ref } from 'vue'

const { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed, saved, save, reset } =
  useSettings()

const {
  containers,
  timers,
  claudeInstances,
  logs,
  logService,
  inferenceStatus,
  inferenceHistory,
  inferenceTools,
  loading,
  error: serverError,
  available,
  refresh,
  fetchLogs,
} = useServerMonitor()

const pollingSeconds = computed({
  get: () => pollingIntervalMs.value / 1000,
  set: (v: number) => {
    pollingIntervalMs.value = v * 1000
  },
})

const serverTab = ref<'containers' | 'logs' | 'claude' | 'timers' | 'inference'>('containers')

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${Math.round(seconds)}s`
  if (seconds < 3600) return `${Math.round(seconds / 60)}m`
  const h = Math.floor(seconds / 3600)
  const m = Math.round((seconds % 3600) / 60)
  return `${h}h ${m}m`
}

function formatDate(dateStr: string): string {
  if (!dateStr) return 'n/a'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

const logServices = ['backend', 'frontend', 'db', 'planner-deploy', 'planner-backup']
</script>

<template>
  <div>
    <PageHeader
      title="Settings"
      subtitle="Configure your preferences"
    />

    <div class="p-6 space-y-8 max-w-2xl">
      <!-- API Configuration -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          API Configuration
        </h2>
        <div>
          <label
            for="api-base-url"
            class="text-sm font-medium text-gray-300 mb-1.5 block"
          >
            API Base URL
          </label>
          <input
            id="api-base-url"
            v-model="apiBaseUrl"
            type="text"
            class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            placeholder="http://localhost:8080"
          >
          <p class="text-xs text-gray-500 mt-1">
            The base URL for the Planner API
          </p>
        </div>
      </div>

      <!-- Display Preferences -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          Display Preferences
        </h2>
        <div class="space-y-4">
          <div>
            <label
              for="polling-interval"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Polling Interval (seconds)
            </label>
            <input
              id="polling-interval"
              v-model.number="pollingSeconds"
              type="number"
              min="5"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              How often to refresh data from the API
            </p>
          </div>

          <div>
            <label
              for="rows-per-page"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Rows Per Page
            </label>
            <input
              id="rows-per-page"
              v-model.number="rowsPerPage"
              type="number"
              min="5"
              max="100"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              Number of items displayed per page in lists
            </p>
          </div>

          <div class="flex items-center justify-between">
            <div>
              <label
                for="sidebar-collapsed"
                class="text-sm font-medium text-gray-300 block"
              >
                Sidebar Collapsed by Default
              </label>
              <p class="text-xs text-gray-500 mt-0.5">
                Start with the sidebar minimized
              </p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                id="sidebar-collapsed"
                v-model="sidebarCollapsed"
                type="checkbox"
                class="sr-only peer"
              >
              <div class="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/50 after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-gray-300 after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
            </label>
          </div>
        </div>
      </div>

      <!-- Server Monitoring -->
      <div v-if="available">
        <div class="flex items-center justify-between mb-4 border-b border-gray-800 pb-2">
          <h2 class="text-base font-semibold text-gray-100">
            Server
          </h2>
          <button
            class="text-gray-400 hover:text-gray-200 transition-colors"
            @click="refresh"
          >
            <svg
              class="w-4 h-4"
              :class="{ 'animate-spin': loading }"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>

        <p
          v-if="serverError"
          class="text-sm text-red-400 mb-3"
        >
          {{ serverError }}
        </p>

        <!-- Tabs -->
        <div class="flex gap-1 mb-4 bg-gray-900 rounded-lg p-1">
          <button
            v-for="tab in (['containers', 'inference', 'logs', 'claude', 'timers'] as const)"
            :key="tab"
            class="flex-1 px-3 py-1.5 text-xs font-medium rounded-md transition-colors capitalize"
            :class="serverTab === tab
              ? 'bg-gray-800 text-gray-100'
              : 'text-gray-400 hover:text-gray-200'"
            @click="serverTab = tab; tab === 'logs' && fetchLogs()"
          >
            {{ tab }}
          </button>
        </div>

        <!-- Containers Tab -->
        <div v-if="serverTab === 'containers'" class="space-y-2">
          <div
            v-for="c in containers"
            :key="c.name"
            class="flex items-center justify-between bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div>
              <p class="text-sm font-medium text-gray-100">{{ c.name }}</p>
              <p class="text-xs text-gray-500">{{ c.image }}</p>
            </div>
            <div class="text-right">
              <span
                class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                :class="c.state === 'running'
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'"
              >
                {{ c.state }}
              </span>
              <p class="text-xs text-gray-500 mt-1">{{ c.status }}</p>
            </div>
          </div>
          <p v-if="containers.length === 0" class="text-sm text-gray-500">No containers found</p>
        </div>

        <!-- Inference Tab -->
        <div v-if="serverTab === 'inference'" class="space-y-4">
          <!-- Current Session -->
          <div class="bg-gray-900 rounded-lg px-4 py-3 border border-gray-800">
            <h3 class="text-sm font-medium text-gray-100 mb-3">Current Session</h3>
            <div v-if="inferenceStatus?.session_id" class="space-y-2">
              <div class="grid grid-cols-2 gap-x-4 gap-y-2 text-xs">
                <div>
                  <span class="text-gray-500">Session</span>
                  <p class="text-gray-300 font-mono truncate">{{ inferenceStatus.session_id }}</p>
                </div>
                <div>
                  <span class="text-gray-500">Age</span>
                  <p class="text-gray-300">{{ formatDuration(inferenceStatus.age_seconds) }}</p>
                </div>
                <div>
                  <span class="text-gray-500">Requests</span>
                  <p class="text-gray-300">{{ inferenceStatus.total_requests }}</p>
                </div>
                <div>
                  <span class="text-gray-500">Avg Duration</span>
                  <p class="text-gray-300">{{ inferenceStatus.avg_duration_ms }}ms</p>
                </div>
              </div>
              <!-- Context usage bar -->
              <div>
                <div class="flex items-center justify-between text-xs mb-1">
                  <span class="text-gray-500">Context Usage</span>
                  <span
                    class="font-medium"
                    :class="inferenceStatus.context_usage_pct > 80
                      ? 'text-red-400'
                      : inferenceStatus.context_usage_pct > 50
                        ? 'text-yellow-400'
                        : 'text-green-400'"
                  >
                    {{ inferenceStatus.context_usage_pct.toFixed(1) }}%
                  </span>
                </div>
                <div class="w-full bg-gray-800 rounded-full h-1.5">
                  <div
                    class="h-1.5 rounded-full transition-all"
                    :class="inferenceStatus.context_usage_pct > 80
                      ? 'bg-red-500'
                      : inferenceStatus.context_usage_pct > 50
                        ? 'bg-yellow-500'
                        : 'bg-green-500'"
                    :style="{ width: Math.min(inferenceStatus.context_usage_pct, 100) + '%' }"
                  />
                </div>
                <p class="text-xs text-gray-500 mt-1">
                  {{ inferenceStatus.latest_input_tokens.toLocaleString() }} / {{ inferenceStatus.context_max.toLocaleString() }} tokens
                </p>
              </div>
            </div>
            <p v-else class="text-sm text-gray-500">No active session</p>
          </div>

          <!-- Session History -->
          <div class="bg-gray-900 rounded-lg px-4 py-3 border border-gray-800">
            <h3 class="text-sm font-medium text-gray-100 mb-3">Session History</h3>
            <div v-if="inferenceHistory.length > 0" class="space-y-2">
              <div
                v-for="s in inferenceHistory.slice().reverse()"
                :key="s.session_id"
                class="flex items-center justify-between text-xs py-1.5 border-b border-gray-800 last:border-0"
              >
                <div>
                  <p class="text-gray-300 font-mono truncate max-w-[180px]">{{ s.session_id.slice(0, 12) }}...</p>
                  <p class="text-gray-500">{{ formatDate(s.created_at) }}</p>
                </div>
                <div class="text-right">
                  <span
                    class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                    :class="{
                      'bg-blue-500/20 text-blue-400': s.end_reason === 'manual',
                      'bg-yellow-500/20 text-yellow-400': s.end_reason === 'context_full',
                      'bg-red-500/20 text-red-400': s.end_reason === 'timeout' || s.end_reason === 'error',
                    }"
                  >
                    {{ s.end_reason }}
                  </span>
                  <p class="text-gray-500 mt-0.5">{{ s.total_requests }} reqs &middot; {{ s.peak_input_tokens.toLocaleString() }} peak tokens</p>
                </div>
              </div>
            </div>
            <p v-else class="text-sm text-gray-500">No past sessions</p>
          </div>

          <!-- Tool Usage -->
          <div v-if="inferenceTools && Object.keys(inferenceTools.tool_frequency).length > 0" class="bg-gray-900 rounded-lg px-4 py-3 border border-gray-800">
            <h3 class="text-sm font-medium text-gray-100 mb-3">Tool Usage</h3>
            <div class="space-y-1.5">
              <div
                v-for="(count, tool) in inferenceTools.tool_frequency"
                :key="tool"
                class="flex items-center justify-between text-xs"
              >
                <span class="text-gray-300 font-mono">{{ tool }}</span>
                <span class="text-gray-400">{{ count }}</span>
              </div>
            </div>
            <p class="text-xs text-gray-500 mt-2">
              Avg {{ inferenceTools.avg_calls_per_request.toFixed(1) }} calls/request
            </p>
          </div>
        </div>

        <!-- Logs Tab -->
        <div v-if="serverTab === 'logs'" class="space-y-3">
          <div class="flex gap-1 flex-wrap">
            <button
              v-for="svc in logServices"
              :key="svc"
              class="px-2.5 py-1 text-xs font-medium rounded-md transition-colors"
              :class="logService === svc
                ? 'bg-blue-600 text-white'
                : 'bg-gray-800 text-gray-400 hover:text-gray-200'"
              @click="fetchLogs(svc)"
            >
              {{ svc }}
            </button>
          </div>
          <pre class="bg-gray-900 border border-gray-800 rounded-lg p-4 text-xs text-gray-300 overflow-x-auto max-h-96 overflow-y-auto font-mono whitespace-pre-wrap">{{ logs || 'No logs available' }}</pre>
        </div>

        <!-- Claude Tab -->
        <div v-if="serverTab === 'claude'" class="space-y-2">
          <div
            v-for="inst in claudeInstances"
            :key="inst.pid"
            class="bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div class="flex items-center justify-between mb-1">
              <span class="text-sm font-medium text-gray-100">PID {{ inst.pid }}</span>
              <div class="flex gap-3 text-xs text-gray-400">
                <span>CPU {{ inst.cpu }}%</span>
                <span>MEM {{ inst.memory }}%</span>
                <span>{{ inst.elapsed }}</span>
              </div>
            </div>
            <p class="text-xs text-gray-500 font-mono truncate">{{ inst.command }}</p>
          </div>
          <p v-if="claudeInstances.length === 0" class="text-sm text-gray-500">No Claude instances running</p>
        </div>

        <!-- Timers Tab -->
        <div v-if="serverTab === 'timers'" class="space-y-2">
          <div
            v-for="t in timers"
            :key="t.name"
            class="flex items-center justify-between bg-gray-900 rounded-lg px-4 py-3 border border-gray-800"
          >
            <div>
              <p class="text-sm font-medium text-gray-100">{{ t.name }}</p>
              <p class="text-xs text-gray-500">Last: {{ t.lastRun || 'never' }}</p>
            </div>
            <div class="text-right">
              <span
                class="inline-block px-2 py-0.5 rounded-full text-xs font-medium"
                :class="t.active
                  ? 'bg-green-500/20 text-green-400'
                  : 'bg-red-500/20 text-red-400'"
              >
                {{ t.active ? 'active' : 'inactive' }}
              </span>
              <p class="text-xs text-gray-500 mt-1">
                {{ t.result === 'success' ? 'Last run OK' : t.result }}
              </p>
            </div>
          </div>
        </div>
      </div>

      <!-- About -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          About
        </h2>
        <div class="space-y-1 text-sm text-gray-400">
          <p>
            <span class="text-gray-300 font-medium">Planner</span> &mdash; Phase 4
          </p>
          <p>Built with Vue 3, Pinia, and Tailwind CSS</p>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-3 pt-2">
        <button
          class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          @click="save"
        >
          Save
        </button>
        <button
          class="bg-gray-800 hover:bg-gray-700 text-gray-300 px-4 py-2 rounded-lg text-sm font-medium border border-gray-700 transition-colors"
          @click="reset"
        >
          Reset to Defaults
        </button>
        <span
          v-if="saved"
          class="text-sm text-green-400 font-medium"
        >
          Saved!
        </span>
      </div>
    </div>
  </div>
</template>
