<script setup lang="ts">
import type { Tag } from '@/types'

defineProps<{
  tag: Tag
  removable?: boolean
}>()

const emit = defineEmits<{
  remove: [id: string]
}>()
</script>

<template>
  <span class="inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-300 border border-gray-700">
    {{ tag.name }}
    <button
      v-if="removable"
      class="ml-0.5 text-gray-500 hover:text-gray-300 transition-colors"
      @click.stop="emit('remove', tag.id)"
    >
      <svg
        class="w-3 h-3"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="2"
          d="M6 18L18 6M6 6l12 12"
        />
      </svg>
    </button>
  </span>
</template>
