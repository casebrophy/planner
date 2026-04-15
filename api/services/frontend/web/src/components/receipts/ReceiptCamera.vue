<script setup lang="ts">
import { ref } from 'vue'

const emit = defineEmits<{
  (e: 'capture', file: File): void
}>()

const previewUrl = ref<string | null>(null)

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  previewUrl.value = URL.createObjectURL(file)
  emit('capture', file)
}
</script>

<template>
  <div class="flex flex-col items-center gap-4">
    <label class="cursor-pointer flex flex-col items-center gap-2 p-6 border-2 border-dashed border-gray-600 rounded-xl hover:border-blue-500 transition-colors">
      <span class="text-4xl">📷</span>
      <span class="text-sm text-gray-400">Tap to capture receipt</span>
      <input
        type="file"
        accept="image/*"
        capture="environment"
        class="hidden"
        @change="handleFileChange"
      >
    </label>
    <img
      v-if="previewUrl"
      :src="previewUrl"
      alt="Receipt preview"
      class="max-h-64 rounded-lg object-contain"
    >
  </div>
</template>
