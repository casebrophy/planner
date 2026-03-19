<script setup lang="ts">
import { useToast } from '@/composables/useToast'

const { toasts, dismiss } = useToast()

function bgClass(type: string): string {
  switch (type) {
    case 'success':
      return 'bg-green-900/90 border-green-700'
    case 'error':
      return 'bg-red-900/90 border-red-700'
    default:
      return 'bg-blue-900/90 border-blue-700'
  }
}
</script>

<template>
  <div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm">
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="px-4 py-3 rounded-lg border shadow-lg text-sm text-gray-100 cursor-pointer"
        :class="bgClass(toast.type)"
        @click="dismiss(toast.id)"
      >
        {{ toast.message }}
      </div>
    </TransitionGroup>
  </div>
</template>

<style scoped>
.toast-enter-active {
  transition: all 0.3s ease;
}
.toast-leave-active {
  transition: all 0.2s ease;
}
.toast-enter-from {
  transform: translateX(100%);
  opacity: 0;
}
.toast-leave-to {
  opacity: 0;
  transform: translateY(10px);
}
</style>
