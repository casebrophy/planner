<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useContextBoard } from '@/composables/useContextBoard'
import { useContextStore } from '@/stores/contextStore'
import PageHeader from '@/components/layout/PageHeader.vue'
import ContextFilterBar from '@/components/contexts/ContextFilterBar.vue'
import ContextKanban from '@/components/contexts/ContextKanban.vue'
import ContextForm from '@/components/contexts/ContextForm.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import type { NewContext, UpdateContext } from '@/types'

const {
  total,
  loading,
  filter,
  contextsByStatus,
  isEmpty,
  setFilter,
  refresh,
} = useContextBoard()

const contextStore = useContextStore()
const router = useRouter()
const showCreateForm = ref(false)

function openContext(id: string) {
  router.push({ name: 'context-detail', params: { id } })
}

async function handleCreate(data: NewContext | UpdateContext) {
  const created = await contextStore.create(data as NewContext)
  showCreateForm.value = false
  if (created) {
    router.push({ name: 'context-detail', params: { id: created.id } })
  }
}
</script>

<template>
  <div>
    <PageHeader
      title="Contexts"
      :subtitle="`${total} contexts`"
    >
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
          New Context
        </button>
      </template>
    </PageHeader>

    <div class="p-6">
      <ContextFilterBar
        :filter="filter"
        class="mb-6"
        @update="setFilter"
      />

      <LoadingSpinner v-if="loading && !contextsByStatus" />

      <EmptyState
        v-else-if="isEmpty"
        title="No contexts found"
        message="Create your first context to organize your work"
        action-label="New Context"
        @action="showCreateForm = true"
      />

      <ContextKanban
        v-else
        :columns="contextsByStatus"
        @select="openContext"
      />
    </div>

    <DrawerPanel
      :open="showCreateForm"
      title="New Context"
      @close="showCreateForm = false"
    >
      <ContextForm
        mode="create"
        @submit="handleCreate"
        @cancel="showCreateForm = false"
      />
    </DrawerPanel>

    <!-- Context detail is full-page via nested route -->
    <router-view />
  </div>
</template>
