<script setup lang="ts">
defineProps<{
  open: boolean
  title?: string
}>()

const emit = defineEmits<{
  close: []
}>()
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer">
      <div
        v-if="open"
        class="fixed inset-0 z-40 flex justify-end"
      >
        <div
          class="absolute inset-0 bg-black/40"
          @click="emit('close')"
        />
        <div class="relative w-full max-w-lg bg-gray-900 border-l border-gray-800 shadow-xl overflow-y-auto">
          <div class="flex items-center justify-between px-6 py-4 border-b border-gray-800">
            <h2 class="text-lg font-semibold text-gray-100">
              {{ title }}
            </h2>
            <button
              class="text-gray-400 hover:text-gray-100 transition-colors"
              @click="emit('close')"
            >
              <svg
                class="w-5 h-5"
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
          </div>
          <div class="p-6">
            <slot />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-enter-active,
.drawer-leave-active {
  transition: transform 0.2s ease, opacity 0.2s ease;
}
.drawer-enter-from,
.drawer-leave-to {
  transform: translateX(100%);
  opacity: 0;
}
</style>
