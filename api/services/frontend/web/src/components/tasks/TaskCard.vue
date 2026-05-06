<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { formatDistanceToNow } from 'date-fns'
import type { Task } from '@/types'
import { TaskStatus, TaskStatusLabels, StatusColors } from '@/types/enums'
import { useContextStore } from '@/stores/contextStore'
import { useTaskStore } from '@/stores/taskStore'
import PriorityIndicator from '@/components/shared/PriorityIndicator.vue'

const props = defineProps<{
  task: Task
}>()

const emit = defineEmits<{
  click: [id: string]
  'status-change': [id: string, status: string]
}>()

const router = useRouter()
const contextStore = useContextStore()
const taskStore = useTaskStore()

const showStatusMenu = ref(false)

const context = computed(() => {
  if (!props.task.contextId) return null
  return contextStore.contextById(props.task.contextId) ?? null
})

const dueLabel = computed(() => {
  if (!props.task.dueDate) return null
  return formatDistanceToNow(new Date(props.task.dueDate), { addSuffix: true })
})

const isOverdue = computed(() => {
  if (!props.task.dueDate) return false
  return new Date(props.task.dueDate) < new Date() && props.task.status !== 'done' && props.task.status !== 'dismissed'
})

const isUnschedulable = computed(() =>
  props.task.status === 'open' && !props.task.scheduledAt && !props.task.dueDate
)

const statusColor = computed(() => StatusColors[props.task.status] ?? '#6b7280')

const statusOptions = [
  TaskStatus.Open,
  TaskStatus.Blocked,
  TaskStatus.Done,
  TaskStatus.Dismissed,
]

function navigateToContext() {
  if (context.value) {
    router.push(`/contexts/${context.value.id}`)
  }
}

async function selectStatus(status: string) {
  showStatusMenu.value = false
  if (status === props.task.status) return
  await taskStore.update(props.task.id, { status: status as TaskStatus })
  emit('status-change', props.task.id, status)
}
</script>

<template>
  <div
    class="bg-gray-900 border border-gray-800 rounded-lg p-4 hover:border-gray-700 cursor-pointer transition-colors"
    @click="emit('click', task.id)"
  >
    <div class="flex items-start justify-between gap-3">
      <h3 class="text-sm font-medium text-gray-100 line-clamp-2">
        {{ task.title }}
      </h3>

      <!-- Inline status selector -->
      <div class="relative flex-shrink-0">
        <button
          class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium transition-opacity hover:opacity-80"
          :style="{ backgroundColor: statusColor + '20', color: statusColor }"
          @click.stop="showStatusMenu = !showStatusMenu"
        >
          <span
            class="w-1.5 h-1.5 rounded-full mr-1.5"
            :style="{ backgroundColor: statusColor }"
          />
          {{ TaskStatusLabels[task.status as keyof typeof TaskStatusLabels] ?? task.status }}
          <svg
            class="ml-1 w-2.5 h-2.5 opacity-60"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>

        <div
          v-if="showStatusMenu"
          class="absolute right-0 top-full mt-1 z-20 bg-gray-800 border border-gray-700 rounded-lg shadow-lg py-1 min-w-[110px]"
          @click.stop
        >
          <button
            v-for="opt in statusOptions"
            :key="opt"
            class="w-full flex items-center gap-2 px-3 py-1.5 text-xs hover:bg-gray-700 transition-colors"
            :class="opt === task.status ? 'opacity-50 cursor-default' : ''"
            @click="selectStatus(opt)"
          >
            <span
              class="w-1.5 h-1.5 rounded-full flex-shrink-0"
              :style="{ backgroundColor: StatusColors[opt] ?? '#6b7280' }"
            />
            <span :style="{ color: StatusColors[opt] ?? '#6b7280' }">{{ TaskStatusLabels[opt] }}</span>
          </button>
        </div>

        <!-- Click-outside backdrop -->
        <div
          v-if="showStatusMenu"
          class="fixed inset-0 z-10"
          @click.stop="showStatusMenu = false"
        />
      </div>
    </div>

    <p
      v-if="task.description"
      class="mt-1.5 text-xs text-gray-500 line-clamp-2"
    >
      {{ task.description }}
    </p>

    <div class="mt-3 flex items-center gap-3 flex-wrap">
      <span
        v-if="task.unconfirmed"
        class="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-amber-900/30 text-amber-400 border border-amber-700/40"
      >
        Unconfirmed
      </span>
      <span
        v-if="task.recurrenceRule"
        class="inline-flex items-center gap-1 text-xs text-blue-400"
        title="Recurring"
      >
        <svg
          class="w-3.5 h-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
          />
        </svg>
      </span>
      <PriorityIndicator :priority="task.priority" />
      <span
        v-if="dueLabel"
        class="text-xs"
        :class="isOverdue ? 'text-red-400' : 'text-gray-500'"
      >
        Due {{ dueLabel }}
      </span>
      <span
        v-if="isUnschedulable"
        class="text-xs text-amber-400/80"
      >
        ⚠ No due date
      </span>
      <button
        v-if="context"
        class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-400 border border-gray-700 hover:text-gray-200 hover:border-gray-600 transition-colors"
        @click.stop="navigateToContext"
      >
        {{ context.title }}
      </button>
    </div>
  </div>
</template>
