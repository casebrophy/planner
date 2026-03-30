<script setup lang="ts">
import { computed } from 'vue'
import { PriorityColors, TaskPriorityLabels } from '@/types/enums'
import type { TaskPriority } from '@/types/enums'

const props = defineProps<{
  priority: TaskPriority
}>()

const color = computed(() => PriorityColors[props.priority] ?? '#6b7280')
const label = computed(() => TaskPriorityLabels[props.priority] ?? props.priority)
</script>

<template>
  <span class="inline-flex items-center gap-1.5 text-xs text-gray-400">
    <span
      class="w-2 h-2 rounded-full"
      :style="{ backgroundColor: color }"
    />
    {{ label }}
  </span>
</template>
