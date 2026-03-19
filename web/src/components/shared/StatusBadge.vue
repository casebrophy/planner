<script setup lang="ts">
import { computed } from 'vue'
import { StatusColors, TaskStatusLabels, ContextStatusLabels } from '@/types/enums'

const props = defineProps<{
  status: string
  type?: 'task' | 'context'
}>()

const label = computed(() => {
  if (props.type === 'context') {
    return ContextStatusLabels[props.status as keyof typeof ContextStatusLabels] ?? props.status
  }
  return TaskStatusLabels[props.status as keyof typeof TaskStatusLabels] ?? props.status
})

const color = computed(() => StatusColors[props.status] ?? '#6b7280')
</script>

<template>
  <span
    class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
    :style="{ backgroundColor: color + '20', color: color }"
  >
    <span class="w-1.5 h-1.5 rounded-full mr-1.5" :style="{ backgroundColor: color }" />
    {{ label }}
  </span>
</template>
