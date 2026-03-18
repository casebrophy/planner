<script setup lang="ts">
import { useCapture } from '@/composables/useCapture'
import PageHeader from '@/components/layout/PageHeader.vue'
import { TaskPriority, TaskEnergy } from '@/types/enums'

const { mode, submitting, taskForm, contextForm, isValid, setMode, submit, reset } = useCapture()
</script>

<template>
  <div>
    <PageHeader title="Quick Capture" subtitle="Quickly create a new task or context" />

    <div class="p-6 max-w-2xl">
      <!-- Mode Toggle -->
      <div class="flex gap-2 mb-6">
        <button
          class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
          :class="mode === 'task' ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400 hover:text-gray-200'"
          @click="setMode('task')"
        >
          Task
        </button>
        <button
          class="px-4 py-2 text-sm font-medium rounded-lg transition-colors"
          :class="mode === 'context' ? 'bg-blue-600 text-white' : 'bg-gray-800 text-gray-400 hover:text-gray-200'"
          @click="setMode('context')"
        >
          Context
        </button>
      </div>

      <!-- Task Form -->
      <form v-if="mode === 'task'" class="space-y-4" @submit.prevent="submit">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Title</label>
          <input
            v-model="taskForm.title"
            type="text"
            class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            placeholder="What needs to be done?"
            autofocus
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Description</label>
          <textarea
            v-model="taskForm.description"
            rows="3"
            class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
            placeholder="Optional details..."
          />
        </div>

        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Priority</label>
            <select
              v-model="taskForm.priority"
              class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            >
              <option :value="TaskPriority.Low">Low</option>
              <option :value="TaskPriority.Medium">Medium</option>
              <option :value="TaskPriority.High">High</option>
              <option :value="TaskPriority.Urgent">Urgent</option>
            </select>
          </div>
          <div>
            <label class="block text-sm font-medium text-gray-300 mb-1">Energy</label>
            <select
              v-model="taskForm.energy"
              class="w-full bg-gray-800 border border-gray-700 text-gray-300 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            >
              <option :value="TaskEnergy.Low">Low</option>
              <option :value="TaskEnergy.Medium">Medium</option>
              <option :value="TaskEnergy.High">High</option>
            </select>
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
            @click="reset"
          >
            Reset
          </button>
          <button
            type="submit"
            :disabled="!isValid || submitting"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ submitting ? 'Creating...' : 'Create Task' }}
          </button>
        </div>
      </form>

      <!-- Context Form -->
      <form v-else class="space-y-4" @submit.prevent="submit">
        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Title</label>
          <input
            v-model="contextForm.title"
            type="text"
            class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500"
            placeholder="Project or area name..."
            autofocus
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-gray-300 mb-1">Description</label>
          <textarea
            v-model="contextForm.description"
            rows="3"
            class="w-full bg-gray-800 border border-gray-700 text-gray-100 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-blue-500 resize-none"
            placeholder="What is this about?"
          />
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            class="px-4 py-2 text-sm font-medium text-gray-300 bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
            @click="reset"
          >
            Reset
          </button>
          <button
            type="submit"
            :disabled="!isValid || submitting"
            class="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-500 rounded-lg transition-colors disabled:opacity-40 disabled:cursor-not-allowed"
          >
            {{ submitting ? 'Creating...' : 'Create Context' }}
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
