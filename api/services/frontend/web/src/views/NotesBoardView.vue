<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useNoteBoard } from '@/composables/useNoteBoard'
import PageHeader from '@/components/layout/PageHeader.vue'
import NoteCard from '@/components/notes/NoteCard.vue'
import NoteFilterBar from '@/components/notes/NoteFilterBar.vue'
import NoteForm from '@/components/notes/NoteForm.vue'
import DrawerPanel from '@/components/shared/DrawerPanel.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import EmptyState from '@/components/shared/EmptyState.vue'
import Pagination from '@/components/shared/Pagination.vue'

const {
  notes,
  total,
  page,
  loading,
  filter,
  isEmpty,
  pagination,
  setFilter,
  setPage,
  refresh,
} = useNoteBoard()

const router = useRouter()
const route = useRoute()

const showCreateForm = ref(false)

const drawerOpen = computed(() => !!route.params.id)

function openNote(id: string) {
  router.push({ name: 'note-detail', params: { id } })
}

function closeDrawer() {
  router.push({ name: 'notes' })
}
</script>

<template>
  <div>
    <PageHeader
      title="Notes"
      :subtitle="`${total} notes`"
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
          New Note
        </button>
      </template>
    </PageHeader>

    <div class="p-6">
      <NoteFilterBar
        :filter="filter"
        class="mb-4"
        @update="setFilter"
      />

      <LoadingSpinner v-if="loading && notes.length === 0" />

      <EmptyState
        v-else-if="isEmpty"
        title="No notes found"
        message="Create your first note to get started"
        action-label="New Note"
        @action="showCreateForm = true"
      />

      <div
        v-else
        class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-3"
      >
        <NoteCard
          v-for="note in notes"
          :key="note.id"
          :note="note"
          @click="openNote"
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
    <DrawerPanel
      :open="showCreateForm"
      title="New Note"
      @close="showCreateForm = false"
    >
      <NoteForm
        mode="create"
        @submit="showCreateForm = false; refresh()"
        @cancel="showCreateForm = false"
      />
    </DrawerPanel>

    <!-- Note Detail Drawer (nested route) -->
    <DrawerPanel
      :open="drawerOpen"
      title="Note Detail"
      @close="closeDrawer"
    >
      <router-view />
    </DrawerPanel>
  </div>
</template>
