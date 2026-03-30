<script setup lang="ts">
import { ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useTaskDetail } from '@/composables/useTaskDetail'
import { useTagStore } from '@/stores/tagStore'
import TaskForm from '@/components/tasks/TaskForm.vue'
import TagList from '@/components/tags/TagList.vue'
import TagPicker from '@/components/tags/TagPicker.vue'
import LoadingSpinner from '@/components/shared/LoadingSpinner.vue'
import ThreadPanel from '@/components/shared/ThreadPanel.vue'
import ConfirmDialog from '@/components/shared/ConfirmDialog.vue'
import type { UpdateTask } from '@/types'

const route = useRoute()
const router = useRouter()
const taskId = route.params.id as string

const { task, tags, loading, update, remove, addTag, removeTag } = useTaskDetail(taskId)
const tagStore = useTagStore()

const editing = ref(false)
const confirmDelete = ref(false)

async function handleUpdate(data: UpdateTask | Record<string, unknown>) {
  await update(data as UpdateTask)
  editing.value = false
}

async function handleDelete() {
  await remove()
  confirmDelete.value = false
  router.push({ name: 'tasks' })
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
</script>

<template>
  <div>
    <LoadingSpinner
      v-if="loading && !task"
      size="sm"
    />

    <div
      v-else-if="task"
      class="space-y-6"
    >
      <!-- View Mode -->
      <div v-if="!editing">
        <div class="flex items-center justify-between mb-4">
          <h3 class="text-lg font-semibold text-gray-100">
            {{ task.title }}
          </h3>
          <div class="flex gap-2">
            <button
              class="px-3 py-1.5 text-sm text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="editing = true"
            >
              Edit
            </button>
            <button
              class="px-3 py-1.5 text-sm text-red-400 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
              @click="confirmDelete = true"
            >
              Delete
            </button>
          </div>
        </div>

        <p
          v-if="task.description"
          class="text-sm text-gray-400 mb-4"
        >
          {{ task.description }}
        </p>

        <div class="space-y-3 text-sm">
          <div class="flex justify-between">
            <span class="text-gray-500">Status</span>
            <span class="text-gray-300">{{ task.status }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Priority</span>
            <span class="text-gray-300">{{ task.priority }}</span>
          </div>
          <div class="flex justify-between">
            <span class="text-gray-500">Energy</span>
            <span class="text-gray-300">{{ task.energy }}</span>
          </div>
          <div
            v-if="task.dueDate"
            class="flex justify-between"
          >
            <span class="text-gray-500">Due</span>
            <span class="text-gray-300">{{ new Date(task.dueDate).toLocaleDateString() }}</span>
          </div>
        </div>

        <!-- Tags -->
        <div class="mt-6">
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

        <!-- Activity Thread -->
        <div class="mt-6">
          <ThreadPanel
            subject-type="task"
            :subject-id="taskId"
          />
        </div>
      </div>

      <!-- Edit Mode -->
      <TaskForm
        v-else
        :task="task"
        mode="edit"
        @submit="handleUpdate"
        @cancel="editing = false"
      />
    </div>

    <ConfirmDialog
      :open="confirmDelete"
      title="Delete Task"
      message="Are you sure you want to delete this task? This action cannot be undone."
      @confirm="handleDelete"
      @cancel="confirmDelete = false"
    />
  </div>
</template>
