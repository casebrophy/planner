<script setup lang="ts">
import { useToday } from '@/composables/useToday'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import { useRouter } from 'vue-router'

const { loading, overdueTasks, dueTodayTasks, inProgressTasks, contextMap, counts, refresh } =
  useToday()
const router = useRouter()

function openTask(id: string) {
  router.push({ name: 'task-detail', params: { id } })
}
</script>

<template>
  <div>
    <PageHeader
      title="Today"
      subtitle="Your tasks for today"
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

    <EmptyState
      v-else-if="counts.overdue === 0 && counts.dueToday === 0 && counts.inProgress === 0"
      title="All clear!"
      message="No tasks due today."
    />

    <div
      v-else
      class="p-6 space-y-6"
    >
      <!-- Overdue -->
      <div v-if="overdueTasks.length > 0">
        <h2 class="text-lg font-semibold text-red-400 mb-3">
          Overdue
          <span class="text-sm font-normal text-gray-400 ml-2">{{ counts.overdue }}</span>
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="task in overdueTasks"
            :key="task.id"
          >
            <TaskCard
              :task="task"
              @click="openTask"
            />
            <span
              v-if="task.contextId && contextMap[task.contextId]"
              class="inline-block mt-1 text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded"
            >
              {{ contextMap[task.contextId] }}
            </span>
          </div>
        </div>
      </div>

      <!-- Due Today -->
      <div v-if="dueTodayTasks.length > 0">
        <h2 class="text-lg font-semibold text-amber-400 mb-3">
          Due Today
          <span class="text-sm font-normal text-gray-400 ml-2">{{ counts.dueToday }}</span>
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="task in dueTodayTasks"
            :key="task.id"
          >
            <TaskCard
              :task="task"
              @click="openTask"
            />
            <span
              v-if="task.contextId && contextMap[task.contextId]"
              class="inline-block mt-1 text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded"
            >
              {{ contextMap[task.contextId] }}
            </span>
          </div>
        </div>
      </div>

      <!-- In Progress -->
      <div v-if="inProgressTasks.length > 0">
        <h2 class="text-lg font-semibold text-blue-400 mb-3">
          In Progress
          <span class="text-sm font-normal text-gray-400 ml-2">{{ counts.inProgress }}</span>
        </h2>
        <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div
            v-for="task in inProgressTasks"
            :key="task.id"
          >
            <TaskCard
              :task="task"
              @click="openTask"
            />
            <span
              v-if="task.contextId && contextMap[task.contextId]"
              class="inline-block mt-1 text-xs text-gray-500 bg-gray-800 px-2 py-0.5 rounded"
            >
              {{ contextMap[task.contextId] }}
            </span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
