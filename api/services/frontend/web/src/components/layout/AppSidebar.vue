<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { useClarificationStore } from '@/stores/clarificationStore'

defineProps<{
  collapsed: boolean
}>()

const emit = defineEmits<{
  toggle: []
}>()

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

const navItems = [
  { name: 'Dashboard', path: '/dashboard', icon: 'grid' },
  { name: 'Today', path: '/today', icon: 'sun' },
  { name: 'Plan', path: '/plan', icon: 'calendar' },
  { name: 'Events', path: '/events', icon: 'event' },
  { name: 'Tasks', path: '/tasks', icon: 'check-square' },
  { name: 'Contexts', path: '/contexts', icon: 'layers' },
  { name: 'Transactions', path: '/transactions', icon: 'credit-card' },
  { name: 'Clarifications', path: '/clarifications', icon: 'alert-circle' },
  { name: 'Capture', path: '/capture', icon: 'plus-circle' },
  { name: 'Search', path: '/search', icon: 'search' },
  { name: 'Settings', path: '/settings', icon: 'settings' },
]

function isActive(path: string): boolean {
  return route.path.startsWith(path)
}
</script>

<template>
  <aside
    class="fixed left-0 top-0 h-full bg-gray-900 border-r border-gray-800 transition-all duration-200 z-40 flex flex-col"
    :class="collapsed ? 'w-16' : 'w-60'"
  >
    <!-- Header -->
    <div class="flex items-center h-14 px-4 border-b border-gray-800">
      <button
        class="text-gray-400 hover:text-gray-100 transition-colors"
        @click="emit('toggle')"
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
            d="M4 6h16M4 12h16M4 18h16"
          />
        </svg>
      </button>
      <span
        v-if="!collapsed"
        class="ml-3 text-lg font-semibold text-gray-100"
      >Planner</span>
    </div>

    <!-- Navigation -->
    <nav class="flex-1 py-4 space-y-1 px-2">
      <router-link
        v-for="item in navItems"
        :key="item.path"
        :to="item.path"
        class="flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors"
        :class="[
          isActive(item.path)
            ? 'bg-gray-800 text-gray-100'
            : 'text-gray-400 hover:text-gray-100 hover:bg-gray-800/50'
        ]"
      >
        <!-- Icons -->
        <svg
          v-if="item.icon === 'grid'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M4 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2V6zM14 6a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2V6zM4 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2H6a2 2 0 01-2-2v-2zM14 16a2 2 0 012-2h2a2 2 0 012 2v2a2 2 0 01-2 2h-2a2 2 0 01-2-2v-2z"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'check-square'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M9 5H7a2 2 0 00-2 2v12a2 2 0 002 2h10a2 2 0 002-2V7a2 2 0 00-2-2h-2M9 5a2 2 0 002 2h2a2 2 0 002-2M9 5a2 2 0 012-2h2a2 2 0 012 2m-6 9l2 2 4-4"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'layers'"
          class="w-5 h-5 shrink-0"
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
        <svg
          v-else-if="item.icon === 'credit-card'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M3 10h18M7 15h1m4 0h1m-7 4h12a3 3 0 003-3V8a3 3 0 00-3-3H6a3 3 0 00-3 3v8a3 3 0 003 3z"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'alert-circle'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'plus-circle'"
          class="w-5 h-5 shrink-0"
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
        <svg
          v-else-if="item.icon === 'sun'"
          class="w-5 h-5 shrink-0"
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
        <svg
          v-else-if="item.icon === 'calendar'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'event'"
          class="w-5 h-5 shrink-0"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            stroke-width="2"
            d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
          />
        </svg>
        <svg
          v-else-if="item.icon === 'search'"
          class="w-5 h-5 shrink-0"
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
        <svg
          v-else-if="item.icon === 'settings'"
          class="w-5 h-5 shrink-0"
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
        <span
          v-if="!collapsed"
          class="ml-3"
        >{{ item.name }}</span>
        <span
          v-if="!collapsed && item.name === 'Clarifications' && clarificationStore.pendingCount > 0"
          class="ml-auto bg-amber-500 text-gray-900 text-xs font-bold px-1.5 py-0.5 rounded-full"
        >
          {{ clarificationStore.pendingCount }}
        </span>
      </router-link>
    </nav>
  </aside>
</template>
