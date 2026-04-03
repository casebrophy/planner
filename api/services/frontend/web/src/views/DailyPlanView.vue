<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useDailyPlan } from '@/composables/useDailyPlan'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import PlanItemCard from '@/components/dailyplan/PlanItemCard.vue'
import PlanGroupHeader from '@/components/dailyplan/PlanGroupHeader.vue'
import { VueDraggable } from 'vue-draggable-plus'

const router = useRouter()
const {
  plan,
  loading,
  generating,
  groupedItems,
  taskMap,
  completedCount,
  totalCount,
  refresh,
  regenerate,
  completeItem,
  dismissItem,
  reorderItem,
} = useDailyPlan()

const dismissingItemId = ref<string | null>(null)
const dismissReason = ref('')
const dismissNote = ref('')

const subtitle = computed(() => {
  if (!plan.value) return 'No plan generated yet'
  return `${completedCount.value}/${totalCount.value} completed`
})

const hasPlan = computed(() => plan.value?.items && plan.value.items.length > 0)

function openTask(taskId: string) {
  const item = plan.value?.items.find((i) => i.id === taskId)
  if (item) {
    router.push({ name: 'task-detail', params: { id: item.taskId } })
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

function handleReorder(groupItems: any[]) {
  for (let i = 0; i < groupItems.length; i++) {
    const item = groupItems[i]
    if ((item.userPosition ?? item.position) !== i) {
      reorderItem(item.id, i)
    }
  }
}
</script>

<template>
  <div>
    <PageHeader
      title="Today's Plan"
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

    <LoadingSpinner v-if="loading && !plan" />

    <div
      v-else
      class="p-6 space-y-6 max-w-4xl"
    >
      <!-- Plan Groups -->
      <template v-if="hasPlan">
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
            item-key="id"
            handle=".cursor-grab"
            class="space-y-2"
            @update:model-value="handleReorder"
          >
            <template #item="{ element: item }">
              <PlanItemCard
                :item="item"
                :task="taskMap[item.taskId]"
                @complete="completeItem(item.id)"
                @dismiss="startDismiss(item.id)"
                @click="openTask(item.id)"
              />
            </template>
          </VueDraggable>
        </div>
      </template>

      <!-- Empty State -->
      <EmptyState
        v-else-if="!loading"
        title="No plan yet"
        message="Generate a plan to see your prioritized tasks for today"
        action-label="Generate Plan"
        @action="regenerate"
      />
    </div>

    <!-- Dismiss Modal -->
    <Teleport
      v-if="dismissingItemId"
      to="body"
    >
      <Transition name="fade">
        <div class="fixed inset-0 z-50 flex items-center justify-center">
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
