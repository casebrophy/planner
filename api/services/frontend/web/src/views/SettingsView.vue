<script setup lang="ts">
import { useSettings } from '@/composables/useSettings'
import PageHeader from '@/components/layout/PageHeader.vue'
import { computed } from 'vue'

const { apiBaseUrl, pollingIntervalMs, rowsPerPage, sidebarCollapsed, saved, save, reset } =
  useSettings()

const pollingSeconds = computed({
  get: () => pollingIntervalMs.value / 1000,
  set: (v: number) => {
    pollingIntervalMs.value = v * 1000
  },
})
</script>

<template>
  <div>
    <PageHeader
      title="Settings"
      subtitle="Configure your preferences"
    />

    <div class="p-6 space-y-8 max-w-2xl">
      <!-- API Configuration -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          API Configuration
        </h2>
        <div>
          <label
            for="api-base-url"
            class="text-sm font-medium text-gray-300 mb-1.5 block"
          >
            API Base URL
          </label>
          <input
            id="api-base-url"
            v-model="apiBaseUrl"
            type="text"
            class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            placeholder="http://localhost:8080"
          >
          <p class="text-xs text-gray-500 mt-1">
            The base URL for the Planner API
          </p>
        </div>
      </div>

      <!-- Display Preferences -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          Display Preferences
        </h2>
        <div class="space-y-4">
          <div>
            <label
              for="polling-interval"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Polling Interval (seconds)
            </label>
            <input
              id="polling-interval"
              v-model.number="pollingSeconds"
              type="number"
              min="5"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              How often to refresh data from the API
            </p>
          </div>

          <div>
            <label
              for="rows-per-page"
              class="text-sm font-medium text-gray-300 mb-1.5 block"
            >
              Rows Per Page
            </label>
            <input
              id="rows-per-page"
              v-model.number="rowsPerPage"
              type="number"
              min="5"
              max="100"
              class="bg-gray-900 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-100 focus:ring-2 focus:ring-blue-500/50 focus:border-blue-500 w-full outline-none"
            >
            <p class="text-xs text-gray-500 mt-1">
              Number of items displayed per page in lists
            </p>
          </div>

          <div class="flex items-center justify-between">
            <div>
              <label
                for="sidebar-collapsed"
                class="text-sm font-medium text-gray-300 block"
              >
                Sidebar Collapsed by Default
              </label>
              <p class="text-xs text-gray-500 mt-0.5">
                Start with the sidebar minimized
              </p>
            </div>
            <label class="relative inline-flex items-center cursor-pointer">
              <input
                id="sidebar-collapsed"
                v-model="sidebarCollapsed"
                type="checkbox"
                class="sr-only peer"
              >
              <div class="w-9 h-5 bg-gray-700 rounded-full peer peer-checked:bg-blue-600 peer-focus:ring-2 peer-focus:ring-blue-500/50 after:content-[''] after:absolute after:top-0.5 after:start-[2px] after:bg-gray-300 after:rounded-full after:h-4 after:w-4 after:transition-all peer-checked:after:translate-x-full" />
            </label>
          </div>
        </div>
      </div>

      <!-- About -->
      <div>
        <h2 class="text-base font-semibold text-gray-100 mb-4 border-b border-gray-800 pb-2">
          About
        </h2>
        <div class="space-y-1 text-sm text-gray-400">
          <p>
            <span class="text-gray-300 font-medium">Planner</span> &mdash; Phase 4
          </p>
          <p>Built with Vue 3, Pinia, and Tailwind CSS</p>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center gap-3 pt-2">
        <button
          class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg text-sm font-medium transition-colors"
          @click="save"
        >
          Save
        </button>
        <button
          class="bg-gray-800 hover:bg-gray-700 text-gray-300 px-4 py-2 rounded-lg text-sm font-medium border border-gray-700 transition-colors"
          @click="reset"
        >
          Reset to Defaults
        </button>
        <span
          v-if="saved"
          class="text-sm text-green-400 font-medium"
        >
          Saved!
        </span>
      </div>
    </div>
  </div>
</template>
