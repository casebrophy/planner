<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import MobileTabBar from './MobileTabBar.vue'
import { useShell } from '@/composables/useShell'
import { ref, onMounted } from 'vue'

const { isMobile } = useShell()

const collapsed = ref(false)

onMounted(() => {
  const saved = localStorage.getItem('sidebar-collapsed')
  if (saved !== null) collapsed.value = saved === 'true'
})

function toggleSidebar() {
  collapsed.value = !collapsed.value
  localStorage.setItem('sidebar-collapsed', String(collapsed.value))
}
</script>

<template>
  <div class="flex h-screen bg-gray-950">
    <!-- Desktop: sidebar -->
    <template v-if="!isMobile">
      <AppSidebar
        :collapsed="collapsed"
        @toggle="toggleSidebar"
      />
      <main
        class="flex-1 overflow-auto"
        :class="collapsed ? 'ml-16' : 'ml-60'"
      >
        <router-view />
      </main>
    </template>

    <!-- Mobile: full-width content + bottom tab bar -->
    <template v-else>
      <main class="flex-1 overflow-auto pb-14">
        <router-view />
      </main>
      <MobileTabBar />
    </template>
  </div>
</template>
