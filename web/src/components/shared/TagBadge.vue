<script setup lang="ts">
import type { Tag } from '@/types'

defineProps<{
  tag: Tag
}>()
</script>

<template>
  <span
    class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-800 text-gray-300 border border-gray-700"
  >
    {{ tag.name }}
  </span>
</template>
