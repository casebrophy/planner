<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()

function goTo(mode: 'task' | 'context' | 'receipt') {
  if (mode === 'receipt') router.push('/receipts')
  else if (mode === 'task') router.push('/tasks')
  else router.push('/contexts')
}
</script>

<template>
  <div class="min-h-screen bg-gray-900">
    <div class="flex flex-col items-center justify-center gap-6 p-6 pt-20">
      <h1 class="text-2xl font-semibold text-white">Capture</h1>
      <p class="text-gray-400 text-center">Choose what you'd like to capture</p>
      <div class="flex gap-3 flex-wrap justify-center">
        <button
          @click="goTo('task')"
          class="px-6 py-3 bg-gray-800 text-white rounded-lg border border-gray-700 hover:bg-gray-700 transition"
        >
          Task
        </button>
        <button
          @click="goTo('context')"
          class="px-6 py-3 bg-gray-800 text-white rounded-lg border border-gray-700 hover:bg-gray-700 transition"
        >
          Context
        </button>
        <button
          @click="goTo('receipt')"
          class="px-6 py-3 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition font-semibold"
        >
          Receipt
        </button>
      </div>
    </div>
  </div>
</template>
