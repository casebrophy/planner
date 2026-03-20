<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
}>()
</script>

<template>
  <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
    <div>
      <h1 class="text-xl font-semibold text-gray-100">
        {{ title }}
      </h1>
      <p
        v-if="subtitle"
        class="text-sm text-gray-400 mt-0.5"
      >
        {{ subtitle }}
      </p>
    </div>
    <div class="flex items-center gap-3">
      <slot name="actions" />
    </div>
  </div>
</template>
