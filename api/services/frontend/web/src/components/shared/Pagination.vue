<script setup lang="ts">
defineProps<{
  page: number
  totalPages: number
  hasNext: boolean
  hasPrev: boolean
}>()

const emit = defineEmits<{
  next: []
  prev: []
  goto: [page: number]
}>()
</script>

<template>
  <div class="flex items-center justify-between px-2 py-3">
    <span class="text-sm text-gray-400">
      Page {{ page }} of {{ totalPages }}
    </span>
    <div class="flex gap-2">
      <button
        :disabled="!hasPrev"
        class="px-3 py-1.5 text-sm rounded-md border border-gray-700 text-gray-300 hover:bg-gray-800 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        @click="emit('prev')"
      >
        Previous
      </button>
      <button
        :disabled="!hasNext"
        class="px-3 py-1.5 text-sm rounded-md border border-gray-700 text-gray-300 hover:bg-gray-800 disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
        @click="emit('next')"
      >
        Next
      </button>
    </div>
  </div>
</template>
