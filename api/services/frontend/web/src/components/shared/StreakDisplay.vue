<script setup lang="ts">
import { computed, onMounted } from 'vue'
import { useActivityLogStore } from '@/stores/activityLogStore'

const props = defineProps<{
  subjectType: string
  subjectId: string
}>()

const store = useActivityLogStore()
const key = computed(() => `${props.subjectType}:${props.subjectId}`)
const streak = computed(() => store.streaks[key.value])

onMounted(() => {
  store.fetchStreaks(props.subjectType, props.subjectId)
})
</script>

<template>
  <div v-if="streak" class="flex items-center gap-4 text-sm">
    <div class="flex items-center gap-1.5">
      <svg class="w-4 h-4 text-orange-400" fill="currentColor" viewBox="0 0 24 24">
        <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
      </svg>
      <span class="text-gray-300">{{ streak.current }}</span>
      <span class="text-gray-500">current</span>
    </div>
    <div class="flex items-center gap-1.5">
      <span class="text-gray-300">{{ streak.longest }}</span>
      <span class="text-gray-500">best</span>
    </div>
    <div class="flex items-center gap-1.5">
      <span class="text-gray-300">{{ streak.totalCount }}</span>
      <span class="text-gray-500">total</span>
    </div>
  </div>
</template>
