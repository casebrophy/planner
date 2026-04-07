<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRawInputStore } from '@/stores/rawinputStore'

const store = useRawInputStore()

let pollInterval: ReturnType<typeof setInterval> | null = null

onMounted(async () => {
  await store.fetchList()
  pollInterval = setInterval(() => store.fetchList(), 15_000)
})

onUnmounted(() => {
  if (pollInterval) clearInterval(pollInterval)
})

const statusOptions = [
  { label: 'All', value: undefined },
  { label: 'Pending', value: 'pending' },
  { label: 'Processing', value: 'processing' },
  { label: 'Processed', value: 'processed' },
  { label: 'Failed', value: 'failed' },
]

function statusBadgeClass(status: string): string {
  const map: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200',
    processing: 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200',
    processed: 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200',
    failed: 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200',
  }
  return map[status] ?? 'bg-gray-100 text-gray-800'
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleString()
}

function isRetryScheduled(item: { status: string; nextRetryAt?: string }): boolean {
  return item.status === 'pending' && !!item.nextRetryAt && new Date(item.nextRetryAt) > new Date()
}
</script>

<template>
  <div class="p-6 space-y-4">
    <div class="flex items-center justify-between">
      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          Ingest Queue
        </h1>
        <p class="text-sm text-gray-500 dark:text-gray-400 mt-0.5">
          {{ store.total }} total · {{ store.failedCount }} failed
        </p>
      </div>
      <button
        class="text-sm px-3 py-1.5 rounded-md bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 text-gray-700 dark:text-gray-200"
        @click="store.fetchList(true)"
      >
        Refresh
      </button>
    </div>

    <div class="flex gap-1.5 flex-wrap">
      <button
        v-for="opt in statusOptions"
        :key="String(opt.value)"
        :class="[
          'px-3 py-1 text-sm rounded-full transition-colors',
          store.statusFilter === opt.value
            ? 'bg-indigo-600 text-white'
            : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-200 hover:bg-gray-200 dark:hover:bg-gray-600',
        ]"
        @click="store.setStatusFilter(opt.value)"
      >
        {{ opt.label }}
      </button>
    </div>

    <div
      v-if="store.loading && store.items.length === 0"
      class="text-center py-12 text-gray-400"
    >
      Loading…
    </div>

    <div
      v-else-if="!store.loading && store.items.length === 0"
      class="text-center py-12 text-gray-400 dark:text-gray-500"
    >
      No raw inputs found.
    </div>

    <div
      v-else
      class="overflow-x-auto rounded-lg border border-gray-200 dark:border-gray-700"
    >
      <table class="w-full text-sm">
        <thead>
          <tr class="bg-gray-50 dark:bg-gray-800 text-left text-gray-600 dark:text-gray-300">
            <th class="px-4 py-3 font-medium">
              Source
            </th>
            <th class="px-4 py-3 font-medium">
              Status
            </th>
            <th class="px-4 py-3 font-medium">
              Retries
            </th>
            <th class="px-4 py-3 font-medium">
              Created
            </th>
            <th class="px-4 py-3 font-medium">
              Error
            </th>
            <th class="px-4 py-3 font-medium" />
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100 dark:divide-gray-700">
          <tr
            v-for="item in store.items"
            :key="item.id"
            class="bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800"
          >
            <td class="px-4 py-3 font-mono text-xs text-gray-500">
              {{ item.sourceType }}
            </td>
            <td class="px-4 py-3">
              <span
                :class="[
                  'inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium',
                  statusBadgeClass(item.status),
                ]"
              >
                {{ item.status }}
              </span>
              <span
                v-if="isRetryScheduled(item)"
                class="ml-1 text-xs text-gray-400"
              >
                (retry {{ formatDate(item.nextRetryAt!) }})
              </span>
            </td>
            <td class="px-4 py-3 text-gray-500">
              {{ item.retryCount }} / {{ item.maxRetries }}
            </td>
            <td class="px-4 py-3 text-gray-500">
              {{ formatDate(item.createdAt) }}
            </td>
            <td class="px-4 py-3 max-w-xs truncate text-red-500 text-xs">
              {{ item.error ?? '—' }}
            </td>
            <td class="px-4 py-3">
              <button
                v-if="item.status === 'failed' || item.status === 'pending'"
                class="text-xs px-2.5 py-1 rounded bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50"
                :disabled="store.loading"
                @click="store.reprocess(item.id)"
              >
                Reprocess
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <div
      v-if="store.totalPages > 1"
      class="flex items-center justify-between text-sm text-gray-500"
    >
      <span>Page {{ store.page }} of {{ store.totalPages }}</span>
      <div class="flex gap-2">
        <button
          :disabled="store.page <= 1"
          class="px-3 py-1 rounded bg-gray-100 dark:bg-gray-700 disabled:opacity-40"
          @click="store.setPage(store.page - 1)"
        >
          Prev
        </button>
        <button
          :disabled="store.page >= store.totalPages"
          class="px-3 py-1 rounded bg-gray-100 dark:bg-gray-700 disabled:opacity-40"
          @click="store.setPage(store.page + 1)"
        >
          Next
        </button>
      </div>
    </div>
  </div>
</template>
