<script setup lang="ts">
import type { Context } from '@/types'
import ContextCard from './ContextCard.vue'

defineProps<{
  columns: Record<string, Context[]>
}>()

const emit = defineEmits<{
  select: [id: string]
}>()

const columnDefs = [
  { key: 'active', label: 'Active', color: 'bg-green-500' },
  { key: 'paused', label: 'Paused', color: 'bg-yellow-500' },
  { key: 'closed', label: 'Closed', color: 'bg-gray-500' },
]
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
    <div v-for="col in columnDefs" :key="col.key" class="flex flex-col">
      <div class="flex items-center gap-2 mb-3">
        <span class="w-2.5 h-2.5 rounded-full" :class="col.color" />
        <h3 class="text-sm font-semibold text-gray-300">{{ col.label }}</h3>
        <span class="text-xs text-gray-500">({{ columns[col.key]?.length ?? 0 }})</span>
      </div>
      <div class="space-y-3 flex-1">
        <ContextCard
          v-for="ctx in columns[col.key]"
          :key="ctx.id"
          :context="ctx"
          @click="emit('select', $event)"
        />
        <div
          v-if="!columns[col.key]?.length"
          class="text-center py-8 text-sm text-gray-600"
        >
          No contexts
        </div>
      </div>
    </div>
  </div>
</template>
