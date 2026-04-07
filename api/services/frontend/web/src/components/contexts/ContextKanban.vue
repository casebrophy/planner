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
  { key: 'project', label: 'Projects', color: 'bg-blue-500' },
  { key: 'area', label: 'Areas', color: 'bg-purple-500' },
]
</script>

<template>
  <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
    <div
      v-for="col in columnDefs"
      :key="col.key"
      class="flex flex-col"
    >
      <div class="flex items-center gap-2 mb-3">
        <span
          class="w-2.5 h-2.5 rounded-full"
          :class="col.color"
        />
        <h3 class="text-sm font-semibold text-gray-300">
          {{ col.label }}
        </h3>
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
          No {{ col.key === 'project' ? 'projects' : 'areas' }} yet
        </div>
      </div>
    </div>
  </div>
</template>
