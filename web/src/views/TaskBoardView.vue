<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useTaskBoard } from '@/composables/useTaskBoard'
import PageHeader from '@/components/layout/PageHeader.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import TaskFilterBar from '@/components/tasks/TaskFilterBar.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import Pagination from '@/components/shared/Pagination.vue'

const {
  tasks,
  total,
  page,
  loading,
  filter,
  isEmpty,
  pagination,
  setFilter,
  setPage,
  refresh,
} = useTaskBoard()

const router = useRouter()
const route = useRoute()

const showCreateForm = ref(false)

const drawerOpen = computed(() => !!route.params.id)

function openTask(id: string) {
  router.push({ name: 'task-detail', params: { id } })
}

function closeDrawer() {
  router.push({ name: 'tasks' })
}
</script>

<template>
  <div>
    <PageHeader title="Tasks" :subtitle="`${total} tasks`">
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg border border-gray-700 transition-colors"
          @click="refresh"
        >
          Refresh
        </button>
        <button
          class="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors"
          @click="showCreateForm = true"
        >
          New Task
        </button>
      </template>
    </PageHeader>

    <div class="p-6">
      <TaskFilterBar :filter="filter" class="mb-4" @update="setFilter" />

      <LoadingSpinner v-if="loading && tasks.length === 0" />

      <EmptyState
        v-else-if="isEmpty"
        title="No tasks found"
        message="Create your first task to get started"
        action-label="New Task"
        @action="showCreateForm = true"
      />

      <div v-else class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3">
        <TaskCard
          v-for="task in tasks"
          :key="task.id"
          :task="task"
          @click="openTask"
        />
      </div>

      <Pagination
        v-if="pagination.totalPages.value > 1"
        :page="page"
        :total-pages="pagination.totalPages.value"
        :has-next="pagination.hasNextPage.value"
        :has-prev="pagination.hasPrevPage.value"
        class="mt-4"
        @next="setPage(page + 1)"
        @prev="setPage(page - 1)"
      />
    </div>

    <!-- Create Form Drawer -->
    <DrawerPanel :open="showCreateForm" title="New Task" @close="showCreateForm = false">
      <TaskForm mode="create" @submit="showCreateForm = false; refresh()" @cancel="showCreateForm = false" />
    </DrawerPanel>

    <!-- Task Detail Drawer (nested route) -->
    <DrawerPanel :open="drawerOpen" title="Task Detail" @close="closeDrawer">
      <router-view />
    </DrawerPanel>
  </div>
</template>
