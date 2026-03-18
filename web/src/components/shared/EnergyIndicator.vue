<script setup lang="ts">
import { computed } from 'vue'
import { TaskEnergyLabels } from '@/types/enums'
import type { TaskEnergy } from '@/types/enums'

const props = defineProps<{
  energy: TaskEnergy
}>()

const label = computed(() => TaskEnergyLabels[props.energy] ?? props.energy)

const bars = computed(() => {
  switch (props.energy) {
    case 'low':
      return 1
    case 'medium':
      return 2
    case 'high':
      return 3
    default:
      return 2
  }
})
</script>

<template>
  <span class="inline-flex items-center gap-1.5 text-xs text-gray-400">
    <span class="flex gap-0.5">
      <span
        v-for="i in 3"
        :key="i"
        class="w-1 h-3 rounded-sm"
        :class="i <= bars ? 'bg-amber-500' : 'bg-gray-700'"
      />
    </span>
    {{ label }}
  </span>
</template>
