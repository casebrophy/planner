<script setup lang="ts">
import { computed } from 'vue'
import { useDashboard } from '@/composables/useDashboard'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'

const { loading, completionTrend, growingBacklogs, repeatedlyDismissed, inactiveContexts, refresh } =
  useDashboard()

// Bar chart: scale bars relative to max value in trend
const maxCompleted = computed(() =>
  Math.max(...completionTrend.value.map((b) => b.completed), 1),
)

function barHeight(count: number): string {
  const pct = Math.round((count / maxCompleted.value) * 100)
  return `${pct}%`
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })
}
</script>

<template>
  <div>
    <PageHeader
      title="Weekly Health"
      subtitle="Patterns and signals from the last 4 weeks"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading" />

    <div
      v-else
      class="p-6 space-y-8"
    >
      <!-- Section: Completion Trend -->
      <section>
        <h2 class="text-xs font-semibold uppercase tracking-widest text-gray-500 mb-4">
          Completion Trend — Last 4 Weeks
        </h2>
        <div
          v-if="completionTrend.every((b) => b.completed === 0)"
          class="text-sm text-gray-600 py-4"
        >
          No task completions recorded in activity logs for this period.
        </div>
        <div
          v-else
          class="flex items-end gap-4 h-28"
        >
          <div
            v-for="bucket in completionTrend"
            :key="bucket.weekLabel"
            class="flex-1 flex flex-col items-center gap-1"
          >
            <span class="text-xs text-gray-400 font-medium">{{ bucket.completed }}</span>
            <div
              class="w-full bg-gray-800 rounded-t relative"
              style="height: 80px; display: flex; align-items: flex-end;"
            >
              <div
                class="w-full bg-indigo-600 rounded-t transition-all"
                :style="{ height: barHeight(bucket.completed) }"
              />
            </div>
            <span class="text-xs text-gray-500 whitespace-nowrap">{{ bucket.weekLabel }}</span>
          </div>
        </div>
      </section>

      <!-- Section: Growing Backlogs -->
      <section>
        <h2 class="text-xs font-semibold uppercase tracking-widest text-gray-500 mb-4">
          Growing Backlogs — Top Areas by Open + Blocked Tasks
        </h2>
        <div
          v-if="growingBacklogs.length === 0"
          class="text-sm text-gray-600 py-4"
        >
          No open or blocked tasks found.
        </div>
        <div
          v-else
          class="space-y-2"
        >
          <div
            v-for="entry in growingBacklogs"
            :key="entry.contextId"
            class="flex items-center gap-3 bg-gray-900 border border-gray-800 rounded-lg px-4 py-3"
          >
            <div class="flex-1 min-w-0">
              <p class="text-sm font-medium text-gray-100 truncate">
                {{ entry.contextTitle }}
              </p>
            </div>
            <div class="flex items-center gap-3 shrink-0 text-xs">
              <span
                v-if="entry.openCount > 0"
                class="text-gray-400"
              >
                {{ entry.openCount }} open
              </span>
              <span
                v-if="entry.blockedCount > 0"
                class="text-amber-400"
              >
                {{ entry.blockedCount }} blocked
              </span>
              <span class="text-gray-300 font-semibold tabular-nums">
                {{ entry.total }} total
              </span>
            </div>
          </div>
        </div>
      </section>

      <!-- Section: Repeatedly Dismissed -->
      <section>
        <h2 class="text-xs font-semibold uppercase tracking-widest text-gray-500 mb-4">
          Repeatedly Dismissed — Tasks You Keep Putting Off
        </h2>
        <div
          v-if="repeatedlyDismissed.length === 0"
          class="text-sm text-gray-600 py-4"
        >
          No dismissed tasks. Clean slate.
        </div>
        <div
          v-else
          class="space-y-2"
        >
          <div
            v-for="task in repeatedlyDismissed"
            :key="task.id"
            class="flex items-center gap-3 bg-gray-900 border border-gray-800 rounded-lg px-4 py-3"
          >
            <div class="flex-1 min-w-0">
              <p class="text-sm text-gray-300 truncate">
                {{ task.title }}
              </p>
            </div>
            <span class="text-xs text-gray-500 shrink-0">
              {{ formatDate(task.updatedAt) }}
            </span>
          </div>
        </div>
      </section>

      <!-- Section: Areas with No Recent Activity -->
      <section>
        <h2 class="text-xs font-semibold uppercase tracking-widest text-gray-500 mb-4">
          Areas with No Recent Activity — Active Contexts Quiet for 14+ Days
        </h2>
        <div
          v-if="inactiveContexts.length === 0"
          class="text-sm text-gray-600 py-4"
        >
          All active contexts have seen recent activity. Good momentum.
        </div>
        <div
          v-else
          class="grid grid-cols-1 md:grid-cols-2 gap-3"
        >
          <div
            v-for="ctx in inactiveContexts"
            :key="ctx.id"
            class="bg-gray-900 border border-gray-800 rounded-lg px-4 py-3"
          >
            <p class="text-sm font-medium text-gray-300">
              {{ ctx.title }}
            </p>
            <p
              v-if="ctx.description"
              class="text-xs text-gray-500 mt-0.5 line-clamp-1"
            >
              {{ ctx.description }}
            </p>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>
