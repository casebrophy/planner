<script setup lang="ts">
import { useSearch } from '@/composables/useSearch'
import PageHeader from '@/components/layout/PageHeader.vue'
import SearchBar from '@/components/shared/SearchBar.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import ContextCard from '@/components/contexts/ContextCard.vue'
import TagBadge from '@/components/tags/TagBadge.vue'
import { useRouter } from 'vue-router'
import { useTaskDrawer } from '@/composables/useTaskDrawer'

const { query, activeTab, loading, hasSearched, filteredTasks, filteredContexts, filteredTags, totalResults, setTab } =
  useSearch()
const router = useRouter()
const { open: openTaskDrawer } = useTaskDrawer()

function openTask(id: string) {
  openTaskDrawer(id)
}

function openContext(id: string) {
  router.push({ name: 'context-detail', params: { id } })
}

const tabs = [
  { key: 'all' as const, label: 'All' },
  { key: 'tasks' as const, label: 'Tasks' },
  { key: 'contexts' as const, label: 'Contexts' },
  { key: 'tags' as const, label: 'Tags' },
]

function tabCount(key: string): number {
  if (key === 'tasks') return filteredTasks.value.length
  if (key === 'contexts') return filteredContexts.value.length
  if (key === 'tags') return filteredTags.value.length
  return totalResults.value
}
</script>

<template>
  <div>
    <PageHeader
      title="Search"
      subtitle="Find tasks, contexts, and tags"
    />

    <div class="p-6 space-y-6">
      <SearchBar v-model="query" />

      <!-- Tab bar -->
      <div class="flex gap-1 bg-gray-900 border border-gray-800 rounded-lg p-1">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="flex-1 px-3 py-2 text-sm font-medium rounded-md transition-colors"
          :class="
            activeTab === tab.key
              ? 'bg-gray-800 text-gray-100 border-b-2 border-blue-500'
              : 'text-gray-400 hover:text-gray-300 hover:bg-gray-800/50'
          "
          @click="setTab(tab.key)"
        >
          {{ tab.label }}
          <span
            v-if="hasSearched"
            class="ml-1 text-xs text-gray-500"
          >
            ({{ tabCount(tab.key) }})
          </span>
        </button>
      </div>

      <!-- Loading -->
      <LoadingSpinner v-if="loading" />

      <!-- Initial state -->
      <div
        v-else-if="!hasSearched"
        class="text-center py-12"
      >
        <p class="text-gray-500 text-sm">
          Type at least 2 characters to search
        </p>
      </div>

      <!-- Empty results -->
      <EmptyState
        v-else-if="totalResults === 0"
        title="No results found"
        :message="`No results found for '${query}'`"
      />

      <!-- Results -->
      <div
        v-else
        class="space-y-6"
      >
        <!-- Tasks section -->
        <div v-if="(activeTab === 'all' || activeTab === 'tasks') && filteredTasks.length > 0">
          <h2 class="text-lg font-semibold text-gray-100 mb-3">
            Tasks
            <span class="text-sm text-gray-500 font-normal ml-2">{{ filteredTasks.length }}</span>
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <TaskCard
              v-for="task in filteredTasks"
              :key="task.id"
              :task="task"
              @click="openTask"
            />
          </div>
        </div>

        <!-- Contexts section -->
        <div v-if="(activeTab === 'all' || activeTab === 'contexts') && filteredContexts.length > 0">
          <h2 class="text-lg font-semibold text-gray-100 mb-3">
            Contexts
            <span class="text-sm text-gray-500 font-normal ml-2">{{ filteredContexts.length }}</span>
          </h2>
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <ContextCard
              v-for="ctx in filteredContexts"
              :key="ctx.id"
              :context="ctx"
              @click="openContext"
            />
          </div>
        </div>

        <!-- Tags section -->
        <div v-if="(activeTab === 'all' || activeTab === 'tags') && filteredTags.length > 0">
          <h2 class="text-lg font-semibold text-gray-100 mb-3">
            Tags
            <span class="text-sm text-gray-500 font-normal ml-2">{{ filteredTags.length }}</span>
          </h2>
          <div class="flex flex-wrap gap-2">
            <TagBadge
              v-for="tag in filteredTags"
              :key="tag.id"
              :tag="tag"
            />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
