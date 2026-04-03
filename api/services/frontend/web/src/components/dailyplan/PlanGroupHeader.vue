<script setup lang="ts">
defineProps<{
  name: string
  itemCount: number
}>()
</script>

<template>
  <div class="flex items-center gap-2 py-2">
    <h3 class="text-sm font-semibold text-gray-300 uppercase tracking-wide">
      {{ name }}
    </h3>
    <span class="text-xs px-1.5 py-0.5 rounded-full bg-gray-700 text-gray-400">{{ itemCount }}</span>
  </div>
</template>
