<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useToday } from '@/composables/useToday'
import { useDailyPlan } from '@/composables/useDailyPlan'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import PlanItemCard from '@/components/dailyplan/PlanItemCard.vue'
import PlanGroupHeader from '@/components/dailyplan/PlanGroupHeader.vue'
import { VueDraggable } from 'vue-draggable-plus'
import type { DailyPlanItem } from '@/types/dailyPlan'

const router = useRouter()

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

const {
  plan,
  loading: loadingPlan,
  generating,
  groupedItems,
  taskMap,
  completedCount,
  totalCount,
  refresh: refreshPlan,
  regenerate,
  completeItem,
  dismissItem,
  reorderItem,
} = useDailyPlan()

const dismissingItemId = ref<string | null>(null)
const dismissReason = ref('')
const dismissNote = ref('')

const hasPlan = computed(() => plan.value?.items && plan.value.items.length > 0)

const subtitle = computed(() => {
  if (hasPlan.value) return `${completedCount.value}/${totalCount.value} completed`
  return 'Your tasks for today'
})

function refresh() {
  refreshToday()
  refreshPlan()
}

function openTask(id: string) {
  if (hasPlan.value) {
    const item = plan.value?.items.find((i) => i.id === id)
    if (item) {
      router.push({ name: 'task-detail', params: { id: item.taskId } })
    }
  } else {
    router.push({ name: 'task-detail', params: { id } })
  }
}

function startDismiss(itemId: string) {
  dismissingItemId.value = itemId
  dismissReason.value = ''
  dismissNote.value = ''
}

function confirmDismiss() {
  if (dismissingItemId.value) {
    dismissItem(dismissingItemId.value, dismissReason.value, dismissNote.value)
    dismissingItemId.value = null
  }
}

function cancelDismiss() {
  dismissingItemId.value = null
  dismissReason.value = ''
  dismissNote.value = ''
}

function handleReorder(groupItems: DailyPlanItem[]) {
  for (let i = 0; i < groupItems.length; i++) {
    const item = groupItems[i]
    if (!item) continue
    if ((item.userPosition ?? item.position) !== i) {
      reorderItem(item.id, i)
    }
  }
}
</script>

<template>
  <div>
    <PageHeader
      title="Today"
      :subtitle="subtitle"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="generating"
          @click="regenerate"
        >
          {{ generating ? 'Generating...' : hasPlan ? 'Regenerate' : 'Generate Plan' }}
        </button>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="(loadingPlan || loadingToday) && !hasPlan" />

    <!-- Plan view -->
    <div
      v-else-if="hasPlan"
      class="p-6 space-y-6 max-w-4xl"
    >
      <div
        v-for="group in groupedItems"
        :key="group.name"
        class="space-y-2"
      >
        <PlanGroupHeader
          :name="group.name"
          :item-count="group.items.length"
        />
        <VueDraggable
          :model-value="group.items"
          group="plan"
          handle=".cursor-grab"
          class="space-y-2"
          @update:model-value="handleReorder"
        >
          <PlanItemCard
            v-for="item in group.items"
            :key="item.id"
            :item="item"
            :task="taskMap[item.taskId]"
            @complete="completeItem(item.id)"
            @dismiss="startDismiss(item.id)"
            @click="openTask(item.id)"
          />
        </VueDraggable>
      </div>
    </div>

    <!-- Urgency fallback (no plan) -->
    <div
      v-else
      class="p-6 space-y-6"
    >
      <!-- Generate Plan banner -->
      <div class="flex items-center justify-between bg-gray-800/60 border border-gray-700 rounded-xl px-5 py-4">
        <div>
          <p class="text-sm font-medium text-gray-200">
            No plan generated yet
          </p>
          <p class="text-xs text-gray-500 mt-0.5">
            Generate an AI-prioritized plan for today
          </p>
        </div>
        <button
          class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          :disabled="generating"
          @click="regenerate"
        >
          {{ generating ? 'Generating...' : 'Generate Plan' }}
        </button>
      </div>

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

    <!-- Dismiss Modal -->
    <Teleport
      v-if="dismissingItemId"
      to="body"
    >
      <Transition name="fade">
        <div
          v-show="true"
          class="fixed inset-0 z-50 flex items-center justify-center"
        >
          <div
            class="absolute inset-0 bg-black/60"
            @click="cancelDismiss"
          />
          <div class="relative bg-gray-900 border border-gray-800 rounded-xl shadow-xl p-6 max-w-sm w-full mx-4">
            <h3 class="text-lg font-semibold text-gray-100">
              Dismiss Task
            </h3>
            <p class="mt-2 text-sm text-gray-400">
              Why are you dismissing this task?
            </p>

            <div class="mt-4 space-y-3">
              <div>
                <label class="block text-sm font-medium text-gray-300 mb-1">Reason</label>
                <select
                  v-model="dismissReason"
                  class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
                >
                  <option value="">
                    Select a reason
                  </option>
                  <option value="completed_elsewhere">
                    Completed elsewhere
                  </option>
                  <option value="not_relevant">
                    Not relevant today
                  </option>
                  <option value="postpone">
                    Need to postpone
                  </option>
                  <option value="blocked">
                    Blocked/depends on other work
                  </option>
                  <option value="other">
                    Other
                  </option>
                </select>
              </div>

              <div>
                <label class="block text-sm font-medium text-gray-300 mb-1">Note (optional)</label>
                <textarea
                  v-model="dismissNote"
                  rows="2"
                  class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
                  placeholder="Add any additional context..."
                />
              </div>
            </div>

            <div class="mt-6 flex justify-end gap-3">
              <button
                class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
                @click="cancelDismiss"
              >
                Cancel
              </button>
              <button
                class="px-4 py-2 text-sm font-medium text-white bg-orange-600 hover:bg-orange-500 rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
                :disabled="!dismissReason"
                @click="confirmDismiss"
              >
                Dismiss
              </button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
