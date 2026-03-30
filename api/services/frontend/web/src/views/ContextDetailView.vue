<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useContextDetail } from '@/composables/useContextDetail'
import { useTagStore } from '@/stores/tagStore'
import ContextForm from '@/components/contexts/ContextForm.vue'
import EventTimeline from '@/components/events/EventTimeline.vue'
import EventForm from '@/components/events/EventForm.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import TaskCard from '@/components/tasks/TaskCard.vue'
import PageHeader from '@/components/layout/PageHeader.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import StatusBadge from '@/components/shared/StatusBadge.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import { observationService, type Observation } from '@/services/observationService'
import type { UpdateContext, NewEvent } from '@/types'

const route = useRoute()
const router = useRouter()
const contextId = route.params.id as string

const observations = ref<Observation[]>([])

onMounted(() => {
  observationService.queryBySubject('context', contextId).then((obs) => {
    observations.value = obs
  })
})

const {
  context,
  events,
  tags,
  linkedTasks,
  loading,
  update,
  remove,
  addEvent,
  addTag,
  removeTag,
} = useContextDetail(contextId)

const tagStore = useTagStore()
const editing = ref(false)
const confirmDelete = ref(false)

async function handleUpdate(data: UpdateContext | Record<string, unknown>) {
  await update(data as UpdateContext)
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'contexts' })
}

async function handleAddEvent(event: NewEvent) {
  await addEvent(event)
}

async function handleAddTag(tagId: string) {
  await addTag(tagId)
}

async function handleCreateTag(name: string) {
  const tag = await tagStore.create({ name })
  if (tag) {
    await addTag(tag.id)
  }
}

function openTask(id: string) {
  router.push({ name: 'task-detail', params: { id } })
}
</script>

<template>
  <div>
    <PageHeader
      :title="context?.title ?? 'Loading...'"
      :subtitle="context?.description"
    >
      <template #actions>
        <button
          class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="router.push({ name: 'contexts' })"
        >
          Back
        </button>
        <button
          v-if="context"
          class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="editing = !editing"
        >
          {{ editing ? 'Cancel' : 'Edit' }}
        </button>
        <button
          v-if="context"
          class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
          @click="confirmDelete = true"
        >
          Delete
        </button>
      </template>
    </PageHeader>

    <LoadingSpinner v-if="loading && !context" />

    <div
      v-else-if="context"
      class="p-6"
    >
      <!-- Edit Form -->
      <div
        v-if="editing"
        class="max-w-2xl mb-8"
      >
        <ContextForm
          :context="context"
          mode="edit"
          @submit="handleUpdate"
          @cancel="editing = false"
        />
      </div>

      <!-- Context Info -->
      <div
        v-else
        class="grid grid-cols-1 lg:grid-cols-3 gap-6"
      >
        <!-- Main Content -->
        <div class="lg:col-span-2 space-y-6">
          <!-- Status & Summary -->
          <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <div class="flex items-center gap-3 mb-3">
              <StatusBadge
                :status="context.status"
                type="context"
              />
            </div>
            <div
              v-if="context.summary"
              class="text-sm text-gray-400"
            >
              {{ context.summary }}
            </div>
          </div>

          <!-- Events -->
          <div>
            <h3 class="text-sm font-semibold text-gray-300 mb-3 uppercase tracking-wider">
              Events
            </h3>
            <EventForm
              class="mb-4"
              @submit="handleAddEvent"
            />
            <EventTimeline :events="events" />
          </div>

          <!-- Activity Thread -->
          <div class="mt-6">
            <ThreadPanel
              subject-type="context"
              :subject-id="contextId"
            />
          </div>

          <!-- Observations -->
          <div
            v-if="observations.length > 0"
            class="mt-6"
          >
            <h4 class="text-sm font-medium text-gray-300 mb-3">
              Observations
            </h4>
            <div class="space-y-2">
              <div
                v-for="obs in observations"
                :key="obs.id"
                class="bg-gray-800 rounded-lg p-3"
              >
                <p class="text-sm text-gray-200">
                  {{ typeof obs.data === 'object' ? JSON.stringify(obs.data) : obs.data }}
                </p>
                <p class="text-xs text-gray-500 mt-1">
                  {{ obs.kind }} · {{ obs.source }}
                </p>
              </div>
            </div>
          </div>
        </div>

        <!-- Sidebar -->
        <div class="space-y-6">
          <!-- Tags -->
          <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <h4 class="text-sm font-medium text-gray-300 mb-2">
              Tags
            </h4>
            <TagList
              :tags="tags"
              removable
              @remove="removeTag"
            />
            <TagPicker
              :selected-ids="tags.map(t => t.id)"
              class="mt-2"
              @add="handleAddTag"
              @create="handleCreateTag"
            />
          </div>

          <!-- Linked Tasks -->
          <div class="bg-gray-900 border border-gray-800 rounded-lg p-4">
            <h4 class="text-sm font-medium text-gray-300 mb-2">
              Linked Tasks ({{ linkedTasks.length }})
            </h4>
            <div class="space-y-2">
              <TaskCard
                v-for="task in linkedTasks"
                :key="task.id"
                :task="task"
                @click="openTask"
              />
            </div>
            <p
              v-if="linkedTasks.length === 0"
              class="text-xs text-gray-500"
            >
              No linked tasks
            </p>
          </div>
        </div>
      </div>
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete Context"
      message="Are you sure you want to delete this context? All linked events will also be deleted."
      @confirm="handleDelete"
      @cancel="confirmDelete = false"
    />
  </div>
</template>
