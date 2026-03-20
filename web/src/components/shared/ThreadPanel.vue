<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { threadService, type ThreadEntry } from '@/services/threadService'
import LoadingSpinner from './LoadingSpinner.vue'

const props = defineProps<{
  subjectType: string
  subjectId: string
}>()

const entries = ref<ThreadEntry[]>([])
const loading = ref(false)

const kindIcons: Record<string, string> = {
  note: 'N',
  status_change: 'S',
  update: 'U',
  decision: 'D',
}

const sourceColors: Record<string, string> = {
  user: '#3b82f6',
  claude: '#8b5cf6',
  system: '#6b7280',
  email: '#f59e0b',
}

async function load() {
  loading.value = true
  try {
    entries.value = await threadService.queryBySubject(props.subjectType, props.subjectId)
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div>
    <h4 class="text-sm font-medium text-gray-300 mb-3">
      Activity
    </h4>

    <LoadingSpinner
      v-if="loading"
      size="sm"
    />

    <div
      v-else-if="entries.length === 0"
      class="text-sm text-gray-500"
    >
      No activity yet.
    </div>

    <div
      v-else
      class="space-y-3"
    >
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex gap-3"
      >
        <!-- Kind indicator -->
        <div
          class="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0"
          :style="{ backgroundColor: (sourceColors[entry.source] ?? '#6b7280') + '22', color: sourceColors[entry.source] ?? '#6b7280' }"
        >
          {{ kindIcons[entry.kind] ?? '?' }}
        </div>

        <div class="flex-1 min-w-0">
          <p class="text-sm text-gray-200">
            {{ entry.content }}
          </p>
          <p class="text-xs text-gray-500 mt-0.5">
            {{ entry.source }} · {{ formatDistanceToNow(new Date(entry.createdAt), { addSuffix: true }) }}
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
