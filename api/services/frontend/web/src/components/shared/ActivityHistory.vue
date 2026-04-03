<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import { activityLogService } from '@/services/activityLogService'
import type { ActivityLog } from '@/types'

const props = defineProps<{
  subjectType: string
  subjectId: string
}>()

const entries = ref<ActivityLog[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

async function load() {
  loading.value = true
  error.value = null
  try {
    const result = await activityLogService.list({
      filter: { subjectType: props.subjectType, subjectId: props.subjectId },
      orderBy: 'logged_at',
      rows: 20,
    })
    entries.value = result.items
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Failed to load activity'
  } finally {
    loading.value = false
  }
}

defineExpose({ reload: load })

onMounted(load)
</script>

<template>
  <div>
    <h4 class="text-sm font-medium text-gray-300 mb-3">
      Activity History
    </h4>
    <div
      v-if="loading"
      class="text-sm text-gray-500"
    >
      Loading...
    </div>
    <div
      v-else-if="error"
      class="text-sm text-red-400"
    >
      {{ error }}
    </div>
    <div
      v-else-if="entries.length === 0"
      class="text-sm text-gray-500"
    >
      No activity logged yet.
    </div>
    <div
      v-else
      class="space-y-2"
    >
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="flex items-center gap-3 text-sm"
      >
        <div class="w-2 h-2 rounded-full bg-blue-500 shrink-0" />
        <span
          v-if="entry.value"
          class="text-gray-300"
        >{{ entry.value }}</span>
        <span
          v-else
          class="text-gray-400 italic"
        >logged</span>
        <span class="text-gray-500 text-xs ml-auto shrink-0">
          {{ formatDistanceToNow(new Date(entry.loggedAt), { addSuffix: true }) }}
        </span>
      </div>
    </div>
  </div>
</template>
