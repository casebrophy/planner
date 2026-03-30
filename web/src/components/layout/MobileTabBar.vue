<script setup lang="ts">
import { useRoute } from 'vue-router'
import { useClarificationStore } from '@/stores/clarificationStore'
import { onMounted, onUnmounted } from 'vue'

const route = useRoute()
const clarificationStore = useClarificationStore()

let countInterval: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  clarificationStore.fetchPendingCount()
  countInterval = setInterval(() => clarificationStore.fetchPendingCount(), 60000)
})

onUnmounted(() => {
  if (countInterval) clearInterval(countInterval)
})

const tabs = [
  { name: 'Capture', path: '/capture', icon: 'plus-circle' },
  { name: 'Today', path: '/today', icon: 'sun' },
  { name: 'Contexts', path: '/contexts', icon: 'layers' },
  { name: 'Search', path: '/search', icon: 'search' },
  { name: 'Settings', path: '/settings', icon: 'settings' },
]

function isActive(path: string): boolean {
  return route.path.startsWith(path)
}
</script>

<template>
  <nav class="fixed bottom-0 left-0 right-0 bg-gray-900 border-t border-gray-800 z-40 pb-[env(safe-area-inset-bottom)]">
    <div class="flex items-center justify-around h-14">
      <router-link
        v-for="tab in tabs"
        :key="tab.path"
        :to="tab.path"
        class="flex flex-col items-center justify-center flex-1 h-full text-xs transition-colors"
        :class="isActive(tab.path) ? 'text-gray-100' : 'text-gray-500'"
      >
        <!-- Capture / plus-circle -->
        <svg
          v-if="tab.icon === 'plus-circle'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 9v3m0 0v3m0-3h3m-3 0H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <!-- Today / sun -->
        <svg
          v-else-if="tab.icon === 'sun'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 3v1m0 16v1m9-9h-1M4 12H3m15.364 6.364l-.707-.707M6.343 6.343l-.707-.707m12.728 0l-.707.707M6.343 17.657l-.707.707M16 12a4 4 0 11-8 0 4 4 0 018 0z"
          />
        </svg>
        <!-- Contexts / layers -->
        <svg
          v-else-if="tab.icon === 'layers'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"
          />
        </svg>
        <!-- Search -->
        <svg
          v-else-if="tab.icon === 'search'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"
          />
        </svg>
        <!-- Settings -->
        <svg
          v-else-if="tab.icon === 'settings'"
          class="w-6 h-6"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.066 2.573c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.573 1.066c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.066-2.573c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z"
          />
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
          />
        </svg>
        <span class="mt-0.5">{{ tab.name }}</span>
      </router-link>
    </div>
  </nav>
</template>
