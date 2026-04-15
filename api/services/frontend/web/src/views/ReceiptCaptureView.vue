<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { useReceiptCaptureStore } from '@/stores/receiptCaptureStore'
import type { ReceiptExtraction } from '@/services/receiptService'
import { useOCR } from '@/composables/useOCR'
import ReceiptCamera from '@/components/receipts/ReceiptCamera.vue'
import ReceiptReview from '@/components/receipts/ReceiptReview.vue'
import SplitEditor from '@/components/receipts/SplitEditor.vue'
import type { NewSplit } from '@/types'

const store = useReceiptCaptureStore()
const { recognize, isProcessing, progress } = useOCR()
const router = useRouter()

const currentStep = computed(() => store.step)

async function handleCapture(file: File) {
  store.setImageFile(file)

  // Run OCR
  const text = await recognize(file)
  store.setOcrText(text)

  // Run extraction
  await store.runExtraction()
}

function handleUpdateExtraction(data: Partial<ReceiptExtraction>) {
  store.updateExtraction(data)
}

function handleConfirmReview() {
  store.confirm()
  router.push('/transactions')
}

function handleStartSplitting() {
  store.startSplitting()
}

async function handleConfirmWithSplits(_splits: NewSplit[]) {
  // In a full implementation, splits would be created via splitService
  // For now, just confirm and navigate back to transactions
  store.confirm()
  router.push('/transactions')
}

function handleCancel() {
  store.reset()
  router.push('/transactions')
}

function handleReviewCancel() {
  store.reset()
}
</script>

<template>
  <div class="min-h-screen bg-gray-900 py-6 px-4">
    <!-- Error banner -->
    <div
      v-if="store.error"
      class="max-w-lg mx-auto mb-4 p-4 bg-red-900 border border-red-700 rounded-lg text-white"
    >
      {{ store.error }}
    </div>

    <div class="max-w-lg mx-auto">
      <!-- Camera / Scanning -->
      <div v-if="currentStep === 'idle' || currentStep === 'scanning'">
        <ReceiptCamera @capture="handleCapture" />
        <button
          class="mt-4 w-full px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white rounded-lg border border-gray-700"
          @click="handleCancel"
        >
          Cancel
        </button>
      </div>

      <!-- Extracting -->
      <div
        v-if="currentStep === 'extracting'"
        class="space-y-4"
      >
        <div class="text-center">
          <p class="text-white mb-4">
            Extracting receipt data...
          </p>
          <div
            v-if="isProcessing"
            class="w-full bg-gray-700 rounded-full h-2"
          >
            <div
              class="bg-blue-600 h-2 rounded-full transition-all duration-300"
              :style="{ width: `${progress}%` }"
            />
          </div>
        </div>
      </div>

      <!-- Reviewing -->
      <div v-if="currentStep === 'reviewing'">
        <ReceiptReview
          v-if="store.extraction"
          :extraction="store.extraction"
          @update="handleUpdateExtraction"
          @confirm="handleConfirmReview"
          @add-splits="handleStartSplitting"
          @cancel="handleReviewCancel"
        />
      </div>

      <!-- Splitting -->
      <div v-if="currentStep === 'splitting'">
        <div class="mb-4">
          <h2 class="text-lg font-semibold text-white">
            Split Expense
          </h2>
        </div>
        <SplitEditor
          :total-cents="store.extraction?.total ?? 0"
          transaction-id="placeholder"
          @done="handleConfirmWithSplits"
        />
        <button
          class="mt-4 w-full px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white rounded-lg border border-gray-700"
          @click="handleCancel"
        >
          Cancel
        </button>
      </div>
    </div>
  </div>
</template>
