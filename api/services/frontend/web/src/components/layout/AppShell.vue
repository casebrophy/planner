<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import { ref, onMounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useMediaQuery } from '@/composables/useMediaQuery'

const collapsed = ref(false)
const isMobile = useMediaQuery('(max-width: 767px)')
const mobileOpen = ref(false)
const route = useRoute()

onMounted(() => {
  const saved = localStorage.getItem('sidebar-collapsed')
  if (saved !== null) collapsed.value = saved === 'true'
})

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem('sidebar-collapsed', String(collapsed.value))
}

// Close mobile drawer on route change
watch(() => route.path, () => {
  if (isMobile.value) mobileOpen.value = false
})

// Reset mobile state when crossing to desktop
watch(isMobile, (mobile) => {
  if (!mobile) mobileOpen.value = false
})
</script>

<template>
  <div class="flex h-screen bg-gray-950">
    <AppSidebar
      :collapsed="collapsed"
      :is-mobile="isMobile"
      :mobile-open="mobileOpen"
      @toggle="toggleSidebar"
      @close-mobile="mobileOpen = false"
    />

    <!-- Mobile hamburger -->
    <button
      v-if="isMobile"
      class="fixed top-3 left-3 z-30 p-2 rounded-lg bg-gray-800 text-gray-400 hover:text-gray-100 transition-colors"
      @click="mobileOpen = true"
    >
      <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16" />
      </svg>
    </button>

    <main
      class="flex-1 overflow-auto transition-[margin] duration-200"
      :class="isMobile ? '' : (collapsed ? 'ml-16' : 'ml-60')"
    >
      <router-view />
    </main>
  </div>
</template>
