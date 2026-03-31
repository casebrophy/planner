<script setup lang="ts">
import { ref } from 'vue'
import { useTransactionStore } from '@/stores/transactionStore'
import { useToastStore } from '@/stores/toastStore'

const transactionStore = useTransactionStore()
const toastStore = useToastStore()

const fileInput = ref<HTMLInputElement | null>(null)
const selectedFile = ref<File | null>(null)
const format = ref('')

const formatOptions = [
  { value: '', label: 'Auto-detect' },
  { value: 'chase_checking', label: 'Chase Checking' },
  { value: 'chase_credit', label: 'Chase Credit Card' },
  { value: 'amex', label: 'American Express' },
]

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] || null
}

async function upload() {
  if (!selectedFile.value) return

  try {
    const result = await transactionStore.importCSV(selectedFile.value, format.value || undefined)
    toastStore.success(`Imported ${result.imported} transactions (${result.skipped} duplicates skipped)`)
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
  } catch (err) {
    toastStore.error(`Import failed: ${err instanceof Error ? err.message : 'Unknown error'}`)
  }
}
</script>

<template>
  <div class="bg-gray-800 rounded-lg p-4 space-y-3">
    <h3 class="text-sm font-medium text-gray-200">
      Import Bank CSV
    </h3>

    <div class="flex items-center gap-3">
      <input
        ref="fileInput"
        type="file"
        accept=".csv"
        class="text-sm text-gray-400 file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:bg-gray-700 file:text-gray-200 hover:file:bg-gray-600"
        @change="onFileChange"
      >

      <select
        v-model="format"
        class="bg-gray-700 text-sm text-gray-200 rounded px-2 py-1.5 border border-gray-600"
      >
        <option
          v-for="opt in formatOptions"
          :key="opt.value"
          :value="opt.value"
        >
          {{ opt.label }}
        </option>
      </select>

      <button
        class="px-3 py-1.5 rounded text-sm font-medium transition-colors"
        :class="
          selectedFile && !transactionStore.importing
            ? 'bg-blue-600 hover:bg-blue-500 text-white'
            : 'bg-gray-700 text-gray-500 cursor-not-allowed'
        "
        :disabled="!selectedFile || transactionStore.importing"
        @click="upload"
      >
        {{ transactionStore.importing ? 'Importing...' : 'Upload' }}
      </button>
    </div>

    <p
      v-if="transactionStore.lastImportResult"
      class="text-xs text-gray-400"
    >
      Last import: {{ transactionStore.lastImportResult.imported }} new,
      {{ transactionStore.lastImportResult.skipped }} skipped of
      {{ transactionStore.lastImportResult.total }} rows
    </p>
  </div>
</template>
