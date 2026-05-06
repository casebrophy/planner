<script setup lang="ts">
import { useToday } from '@/composables/useToday'
import { useTaskDrawer } from '@/composables/useTaskDrawer'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'

const { open: openTaskDrawer } = useTaskDrawer()

const {
  loading: loadingToday,
  overdueTasks,
  dueTodayTasks,
  blockedTasks,
  unschedulableTasks,
  contextMap,
  counts,
  refresh: refreshToday,
} = useToday()

function openTask(id: string) {
  openTaskDrawer(id)
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
          @click="refreshToday"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loadingToday" />

    <div
      v-else
      class="p-6 space-y-6"
    >
      <EmptyState
        v-if="counts.overdue === 0 && counts.dueToday === 0 && counts.blocked === 0 && counts.unschedulable === 0"
        title="All clear!"
        message="No tasks due today."
      />

      <template v-else>
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

        <!-- Blocked -->
        <div v-if="blockedTasks.length > 0">
          <h2 class="text-lg font-semibold text-blue-400 mb-3">
            Blocked
            <span class="text-sm font-normal text-gray-400 ml-2">{{ counts.blocked }}</span>
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="task in blockedTasks"
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

        <!-- Needs Classification -->
        <div v-if="unschedulableTasks.length > 0">
          <h2 class="text-lg font-semibold text-yellow-500 mb-3">
            Needs Classification
            <span class="text-sm font-normal text-gray-400 ml-2">{{ counts.unschedulable }}</span>
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div
              v-for="task in unschedulableTasks"
              :key="task.id"
            >
              <TaskCard
                :task="task"
                @click="openTask"
              />
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>
