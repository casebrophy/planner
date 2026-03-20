<script setup lang="ts">
import { computed } from 'vue'
import { formatDistanceToNow } from 'date-fns'
import type { Context } from '@/types'
import StatusBadge from '@/components/shared/StatusBadge.vue'

const props = defineProps<{
  context: Context
}>()

const emit = defineEmits<{
  click: [id: string]
}>()

const lastEventLabel = computed(() => {
  if (!props.context.lastEvent) return null
  return formatDistanceToNow(new Date(props.context.lastEvent), { addSuffix: true })
})
</script>

<template>
  <div
    class="bg-gray-900 border border-gray-800 rounded-lg p-4 hover:border-gray-700 cursor-pointer transition-colors"
    @click="emit('click', context.id)"
  >
    <div class="flex items-start justify-between gap-3">
      <h3 class="text-sm font-medium text-gray-100 line-clamp-1">
        {{ context.title }}
      </h3>
      <StatusBadge
        :status="context.status"
        type="context"
      />
    </div>

    <p
      v-if="context.description"
      class="mt-1.5 text-xs text-gray-500 line-clamp-2"
    >
      {{ context.description }}
    </p>

    <div
      v-if="context.summary"
      class="mt-2 text-xs text-gray-400 line-clamp-2 italic"
    >
      {{ context.summary }}
    </div>

    <div class="mt-3 text-xs text-gray-500">
      <span v-if="lastEventLabel">Last event {{ lastEventLabel }}</span>
      <span v-else>No events yet</span>
    </div>
  </div>
</template>
