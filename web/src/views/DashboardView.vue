<script setup lang="ts">
import { useDashboard } from '@/composables/useDashboard'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { useRouter } from 'vue-router'

const { loading, taskCounts, contextCounts, recentTasks, overdueTasks, activeContexts, refresh } =
  useDashboard()
const router = useRouter()

function openTask(id: string) {
  router.push({ name: 'task-detail', params: { id } })
}
</script>

<template>
  <div>
    <PageHeader
      title="Dashboard"
      subtitle="Overview of your tasks and contexts"
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
      class="p-6 space-y-6"
    >
      <!-- Summary Cards -->
      <div class="grid grid-cols-2 md:grid-cols-4 gap-4">
        <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <p class="text-sm text-gray-400">
            Total Tasks
          </p>
          <p class="text-2xl font-bold text-gray-100 mt-1">
            {{ taskCounts.total }}
          </p>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <p class="text-sm text-gray-400">
            In Progress
          </p>
          <p class="text-2xl font-bold text-blue-400 mt-1">
            {{ taskCounts.inProgress }}
          </p>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <p class="text-sm text-gray-400">
            Overdue
          </p>
          <p
            class="text-2xl font-bold mt-1"
            :class="taskCounts.overdue > 0 ? 'text-red-400' : 'text-gray-100'"
          >
            {{ taskCounts.overdue }}
          </p>
        </div>
        <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
          <p class="text-sm text-gray-400">
            Active Contexts
          </p>
          <p class="text-2xl font-bold text-green-400 mt-1">
            {{ contextCounts.active }}
          </p>
        </div>
      </div>

      <!-- Overdue Tasks -->
      <div v-if="overdueTasks.length > 0">
        <h2 class="text-lg font-semibold text-gray-100 mb-3">
          Overdue Tasks
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <TaskCard
            v-for="task in overdueTasks"
            :key="task.id"
            :task="task"
            @click="openTask"
          />
        </div>
      </div>

      <!-- Recent Tasks -->
      <div>
        <h2 class="text-lg font-semibold text-gray-100 mb-3">
          Recent Tasks
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <TaskCard
            v-for="task in recentTasks"
            :key="task.id"
            :task="task"
            @click="openTask"
          />
        </div>
        <div
          v-if="recentTasks.length === 0"
          class="text-sm text-gray-500 py-4"
        >
          No tasks yet
        </div>
      </div>

      <!-- Active Contexts -->
      <div>
        <h2 class="text-lg font-semibold text-gray-100 mb-3">
          Active Contexts
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
          <div
            v-for="ctx in activeContexts"
            :key="ctx.id"
            class="bg-gray-900 border border-gray-800 rounded-lg p-4 cursor-pointer hover:border-gray-700 transition-colors"
            @click="router.push({ name: 'context-detail', params: { id: ctx.id } })"
          >
            <h3 class="text-sm font-medium text-gray-100">
              {{ ctx.title }}
            </h3>
            <p
              v-if="ctx.description"
              class="text-xs text-gray-500 mt-1 line-clamp-2"
            >
              {{ ctx.description }}
            </p>
          </div>
        </div>
        <div
          v-if="activeContexts.length === 0"
          class="text-sm text-gray-500 py-4"
        >
          No active contexts
        </div>
      </div>
    </div>
  </div>
</template>
