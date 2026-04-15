import { defineStore } from 'pinia'
import { ref } from 'vue'
import { receiptService } from '@/services/receiptService'
import type { ReceiptExtraction } from '@/services/receiptService'
import type { NewSplit } from '@/types'

export type CaptureStep = 'idle' | 'scanning' | 'extracting' | 'reviewing' | 'splitting' | 'confirmed'

export interface ReceiptCaptureState {
  step: CaptureStep
  imageFile: File | null
  ocrText: string
  extraction: ReceiptExtraction | null
  splits: NewSplit[]
  error: string | null
}

export const useReceiptCaptureStore = defineStore('receiptCapture', () => {
  const step = ref<CaptureStep>('idle')
  const imageFile = ref<File | null>(null)
  const ocrText = ref<string>('')
  const extraction = ref<ReceiptExtraction | null>(null)
  const splits = ref<NewSplit[]>([])
  const error = ref<string | null>(null)

  const reset = () => {
    step.value = 'idle'
    imageFile.value = null
    ocrText.value = ''
    extraction.value = null
    splits.value = []
    error.value = null
  }

  const setImageFile = (file: File) => {
    imageFile.value = file
    step.value = 'scanning'
    error.value = null
  }

  const setOcrText = (text: string) => {
    ocrText.value = text
    step.value = 'extracting'
    error.value = null
  }

  const runExtraction = async () => {
    try {
      step.value = 'extracting'
      error.value = null
      const result = await receiptService.extractReceipt(ocrText.value)
      extraction.value = result
      step.value = 'reviewing'
    } catch (err) {
      error.value = err instanceof Error ? err.message : 'Failed to extract receipt data'
      step.value = 'reviewing'
    }
  }

  const updateExtraction = (updates: Partial<ReceiptExtraction>) => {
    if (extraction.value) {
      extraction.value = {
        ...extraction.value,
        ...updates,
      }
    }
  }

  const startSplitting = () => {
    step.value = 'splitting'
    error.value = null
  }

  const addSplit = (split: NewSplit) => {
    splits.value.push(split)
  }

  const removeSplit = (index: number) => {
    splits.value.splice(index, 1)
  }

  const updateSplit = (index: number, updates: Partial<NewSplit>) => {
    if (index >= 0 && index < splits.value.length && splits.value[index]) {
      const existing = splits.value[index]!
      splits.value[index] = {
        transactionId: updates.transactionId ?? existing.transactionId,
        partyName: updates.partyName ?? existing.partyName,
        amount: updates.amount ?? existing.amount,
        venmoHandle: updates.venmoHandle ?? existing.venmoHandle,
      }
    }
  }

  const confirm = () => {
    step.value = 'confirmed'
  }

  return {
    step,
    imageFile,
    ocrText,
    extraction,
    splits,
    error,
    reset,
    setImageFile,
    setOcrText,
    runExtraction,
    updateExtraction,
    startSplitting,
    addSplit,
    removeSplit,
    updateSplit,
    confirm,
  }
})
